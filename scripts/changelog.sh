#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'EOF'
usage:
  scripts/changelog.sh update                       write this PR's section into CHANGELOG.md (run via 'make changelog')
  scripts/changelog.sh check                        CI: fail unless the PR adds the expected CHANGELOG.md section
  scripts/changelog.sh release-plan                 CI: find merged CHANGELOG.md sections with no GitHub release
  scripts/changelog.sh tag <version> <commit>       CI: tag <commit> as v<version> and push the tag
  scripts/changelog.sh notes <from> <to> [entry]    print the release notes for <from>..<to>, plus an extra entry
  scripts/changelog.sh extract <version>            print the <version> section body from CHANGELOG.md
  scripts/changelog.sh release-notes <tag>          CI: print the GitHub release body for <tag> (section, or generated notes)

labels: no-release (no section, no release), release:minor, release:major (default: patch)
EOF
	exit 2
}

die() {
	echo "changelog: $*" >&2
	exit 1
}

semver='^[0-9]+\.[0-9]+\.[0-9]+$'
base_branch=${BASE_BRANCH:-master}

out() {
	local key=$1 value=$2
	echo "$key=$value" >>"${GITHUB_OUTPUT:-/dev/stdout}"
}

summary() {
	local text=$1
	echo "$text"
	if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
		echo "$text" >>"$GITHUB_STEP_SUMMARY"
	fi
}

fail() {
	local message=$1
	echo "::error title=CHANGELOG.md::$message"
	summary "### ❌ CHANGELOG.md check failed"
	summary ""
	summary "$message"
	summary ""
	summary "Fix: on your branch run \`make changelog\`, commit CHANGELOG.md and push. Add the \`no-release\` label if this PR should not be released."
	exit 1
}

tag_exists() {
	local version=$1
	git rev-parse -q --verify "refs/tags/v$version" >/dev/null
}

release_tags() {
	local ref=$1
	git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname --merged "$ref"
}

latest_tag() {
	local ref=$1
	release_tags "$ref" | sed -n 1p
}

version_gt() {
	local a=$1 b=$2
	[[ $a != "$b" && $(printf '%s\n%s\n' "$a" "$b" | sort -V | tail -1) == "$a" ]]
}

# A release heading ends with a year, which skips master's old
# "## 0.1.0 (Unreleased)" stub and "## 0.0.54 and earlier".
versions_in() {
	sed -nE 's/^## ([0-9]+\.[0-9]+\.[0-9]+) \(.*[0-9]{4}\)$/\1/p'
}

release_date() {
	local rev=${1:-}
	if [[ -z $rev ]]; then
		LC_ALL=C date '+%B %-d, %Y'
	else
		LC_ALL=C git log -1 --format=%cd --date=format:'%B %-d, %Y' "$rev"
	fi
}

# The newest version on ref: its latest tag, or a merged section that is not
# tagged yet because its release is still running.
current_version() {
	local ref=$1 tag section
	tag=$(latest_tag "$ref")
	section=$(git show "$ref:CHANGELOG.md" | versions_in | sort -V | tail -1)
	printf '%s\n%s\n' "${tag#v}" "$section" | sed '/^$/d' | sort -V | tail -1
}

# The newest commit that added or removed the heading is the one that added
# the heading present now; a later edit (e.g. the date) keeps the count.
section_commit() {
	local ref=$1 version=$2 since=$3
	git log --format=%H -S "## $version (" "$since..$ref" -- CHANGELOG.md | sed -n 1p
}

range_start() {
	local ref=$1 version=$2 tag sha
	tag=$(latest_tag "$ref")
	if [[ $version == "${tag#v}" ]]; then
		echo "$tag"
		return
	fi
	sha=$(section_commit "$ref" "$version" "$tag")
	[[ -n $sha ]] || die "cannot find the commit that added the $version section"
	echo "$sha"
}

has_label() {
	local labels=$1 label=$2
	[[ ",$labels," == *",$label,"* ]]
}

next_version() {
	local last=$1 labels=$2 major minor patch
	IFS=. read -r major minor patch <<<"${last#v}"
	if has_label "$labels" release:major; then
		echo "$((major + 1)).0.0"
	elif has_label "$labels" release:minor; then
		echo "$major.$((minor + 1)).0"
	else
		echo "$major.$minor.$((patch + 1))"
	fi
}

repo_url() {
	local url
	url=$(git remote get-url origin)
	url=${url%.git}
	url=${url/#git@github.com:/https://github.com/}
	echo "$url"
}

notes() {
	local from=$1 to=$2 entry=${3:-} body
	body=$(
		{
			[[ -z $entry ]] || printf -- '- %s\n' "$entry"
			git log --format='%h %s' --abbrev=7 "$from..$to"
		} | REPO_URL=$(repo_url) perl -ne '
			chomp;
			my ($sha, $msg) = /^(\S+) (.*)$/ or next;
			next if $msg =~ /Merge (pull request|branch|remote-tracking branch)/;
			my $g = $msg =~ /^.*?feat(\([^)]+\))?!?:.+$/ ? 0
				: $msg =~ /(?i)^.*?fix(\([^)]+\))?!?:.+$/ ? 1
				: $msg =~ /(?i)^.*?(chore\(deps\)|bump \S+ from \S+ to \S+)/ ? 3
				: 2;
			if ($msg =~ /\(#\d+\)/) {
				$msg =~ s{\(#(\d+)\)}{([#$1]($ENV{REPO_URL}/pull/$1))}g;
			} elsif ($sha ne "-") {
				$msg .= " ([$sha]($ENV{REPO_URL}/commit/$sha))";
			}
			push @{ $groups{$g} }, "* $msg";
			END {
				my @titles = ("Features", "Bug fixes", "Changes", "Dependencies");
				my $n = 0;
				for my $i (0 .. 3) {
					next unless $groups{$i};
					print "\n" if $n++;
					print "### $titles[$i]\n\n", map { "$_\n" } @{ $groups{$i} };
				}
			}
		'
	)
	echo "${body:-No user-facing changes.}"
}

extract() {
	local version=$1
	awk -v v="$version" '
		index($0, "## " v " ") == 1 || $0 == "## " v { found = 1; next }
		found && /^## / { exit }
		found { print }
		END { if (!found) exit 1 }
	' CHANGELOG.md | perl -0777 -pe 's/\A\s+//; s/\s+\z/\n/'
}

release_notes() {
	local tag=$1 version=${1#v} previous
	if ! [[ $version =~ $semver ]] || ! tag_exists "$version"; then
		die "release-notes: $tag is not a release tag; run release.yml on a v* tag ref"
	fi
	if extract "$version"; then
		return
	fi
	previous=$(latest_tag "v$version^")
	[[ -n $previous ]] || die "release-notes: no v* tag before v$version to generate the notes from"
	echo "::warning::CHANGELOG.md has no section for v$version; the notes were generated from the commits since $previous. The next \`make changelog\` backfills this version into CHANGELOG.md." >&2
	notes "$previous" "v$version"
}

generate() {
	local version=$1 date=$2 from=$3 to=$4 entry=${5:-} body section out
	if extract "$version" >/dev/null 2>&1; then
		die "CHANGELOG.md already has a section for $version"
	fi
	body=$(notes "$from" "$to" "$entry")
	section=$(mktemp)
	out=$(mktemp)
	printf '## %s (%s)\n\n%s\n\n' "$version" "$date" "$body" >"$section"
	awk -v section="$section" '
		!done && /^## / { while ((getline line < section) > 0) print line; done = 1 }
		{ print }
		END { if (!done) while ((getline line < section) > 0) print line }
	' CHANGELOG.md >"$out"
	cp "$out" CHANGELOG.md
	rm -f "$section" "$out"
	echo "CHANGELOG.md: added $version"
}

remove_section() {
	local file=$1 version=$2 tmp
	tmp=$(mktemp)
	awk -v v="$version" '
		index($0, "## " v " ") == 1 { skip = 1; next }
		skip && /^## / { skip = 0 }
		!skip { print }
	' "$file" >"$tmp"
	cp "$tmp" "$file"
	rm -f "$tmp"
}

untagged_new_versions() {
	local head_file=$1 base_file=$2 v
	for v in $(versions_in <"$head_file"); do
		if ! tag_exists "$v" && ! grep -qx "$v" <(versions_in <"$base_file"); then
			echo "$v"
		fi
	done
}

backfill_missing_releases() {
	local base=$1 newest previous="" t
	newest=$(versions_in <CHANGELOG.md | sort -V | tail -1)
	[[ -n $newest ]] || return 0
	for t in $(release_tags "$base" | sort -V); do
		if [[ -n $previous ]] && version_gt "${t#v}" "$newest"; then
			generate "${t#v}" "$(release_date "$t")" "$previous" "$t"
		fi
		previous=$t
	done
}

update() {
	local pr number title labels base base_file last version from v
	local -a selector=()
	command -v gh >/dev/null || die "gh CLI is required to read the PR title and labels"
	[[ -z ${PR_NUMBER:-} ]] || selector=("$PR_NUMBER")
	pr=$(gh pr view ${selector[@]+"${selector[@]}"} --json number,title,labels --jq '[.number, .title, ([.labels[].name] | join(","))] | @tsv') ||
		die "no PR found for this branch; open it first (a draft is fine), then rerun 'make changelog'"
	IFS=$'\t' read -r number title labels <<<"$pr"

	git fetch -q origin "$base_branch" --tags
	base=origin/$base_branch
	if ! git diff --quiet "$(git merge-base HEAD "$base")" "$base" -- CHANGELOG.md; then
		die "CHANGELOG.md changed on $base_branch since this branch started (another PR was released); merge or rebase on $base_branch, then rerun 'make changelog'"
	fi
	base_file=$(mktemp)
	git show "$base:CHANGELOG.md" >"$base_file"
	for v in $(untagged_new_versions CHANGELOG.md "$base_file"); do
		remove_section CHANGELOG.md "$v"
	done
	rm -f "$base_file"
	backfill_missing_releases "$base"

	if has_label "$labels" no-release; then
		echo "PR #$number has the no-release label: no section added (this change ships with the next release)"
		return
	fi

	last=$(current_version "$base")
	[[ -n $last ]] || die "no v* tag found on $base"
	version=$(next_version "$last" "$labels")
	from=$(range_start "$base" "$last")
	generate "$version" "$(release_date)" "$from" "$base" "$title (#$number)"
	echo "PR #$number releases v$version on merge. Review the section, then commit CHANGELOG.md."
}

check() {
	local number=${PR_NUMBER:?} title=${PR_TITLE:?} labels=${PR_LABELS:-} author=${PR_AUTHOR:-}
	local base=${BASE_REF:-HEAD^1} base_file last expected version body missing ref from v
	local -a new=()
	[[ $number =~ ^[0-9]+$ ]] || die "invalid PR number '$number'"

	if [[ $author == "dependabot[bot]" ]]; then
		summary "✅ Dependabot PR: exempt. Its bump is listed in the next release's section."
		return
	fi

	base_file=$(mktemp)
	git show "$base:CHANGELOG.md" >"$base_file"
	while IFS= read -r v; do
		new+=("$v")
	done < <(untagged_new_versions CHANGELOG.md "$base_file")
	rm -f "$base_file"

	if has_label "$labels" no-release; then
		((${#new[@]} == 0)) || fail "This PR has the \`no-release\` label but adds a section for ${new[*]}. Remove the section or the label."
		summary "✅ \`no-release\` label: no CHANGELOG.md section needed. This change ships with the next release."
		return
	fi

	last=$(current_version "$base")
	[[ -n $last ]] || die "no v* tag found on $base"
	expected=$(next_version "$last" "$labels")
	((${#new[@]} > 0)) || fail "This PR does not add a CHANGELOG.md section. Merging it releases **v$expected**, so it needs a \`## $expected (...)\` section."
	((${#new[@]} == 1)) || fail "This PR adds more than one new section (${new[*]}); it should add only \`## $expected\`."
	version=${new[0]}
	[[ $version == "$expected" ]] || fail "The new section is \`## $version\`, but merging this PR releases **v$expected** (latest version on $base_branch: $last, labels: ${labels:-none})."

	body=$(extract "$version")
	[[ $body == *"#$number"* ]] || fail "The \`## $version\` section has no entry for this PR (#$number)."

	from=$(range_start "$base" "$last")
	missing=""
	for ref in $(notes "$from" "$base" "$title (#$number)" | grep -oE '\[#[0-9]+\]' | tr -d '[]' | sort -u); do
		[[ $body == *"$ref"* ]] || missing+=" $ref"
	done
	[[ -z $missing ]] || fail "The \`## $version\` section is missing changes merged since $last:$missing. Rerun \`make changelog\` to regenerate it."

	summary "### ✅ CHANGELOG.md: merging this PR releases v$version"
	summary ""
	summary "$body"
}

merged_pr_labels() {
	local sha=$1
	gh api "repos/{owner}/{repo}/commits/$sha/pulls" --jq 'if length == 0 then error("no PR") else [.[0].labels[].name] | join(",") end'
}

# A version is pending until its GitHub release exists: release.yml pushes the
# tag before GoReleaser publishes, so a tag alone may be a failed release.
# Every pending merged section is released, oldest first: when several PRs
# merge before a run starts, GitHub replaces the queued runs with one.
# An untagged version must be exactly what its PR's labels allow, as the PR
# check requires, because that check is not a required status check. An
# existing tag already fixes the version and the commit.
release_plan() {
	local published last="" t v sha previous labels expected releases="[]"
	published=$(gh release list --exclude-drafts --limit 1000 --json tagName --jq '.[].tagName') ||
		die "cannot list GitHub releases; nothing was released"
	for t in $(release_tags HEAD); do
		if grep -qxF "$t" <<<"$published"; then
			last=$t
			break
		fi
	done
	[[ -n $last ]] || die "no released v* tag found"
	previous=${last#v}
	for v in $(versions_in <CHANGELOG.md | sort -V); do
		version_gt "$v" "${last#v}" || continue
		grep -qxF "v$v" <<<"$published" && continue
		if tag_exists "$v"; then
			sha=$(git rev-parse "v$v^{commit}")
		else
			sha=$(section_commit HEAD "$v" "$last")
			[[ -n $sha ]] || die "cannot find the commit that added the $v section"
			labels=$(merged_pr_labels "$sha") || die "cannot read the PR labels for ${sha:0:7}; nothing was released"
			expected=$(next_version "$previous" "$labels")
			[[ $v == "$expected" ]] ||
				die "CHANGELOG.md section $v (${sha:0:7}) should be $expected after $previous (PR labels: ${labels:-none}); nothing was released"
		fi
		previous=$v
		echo "release v$v at ${sha:0:7}"
		releases=$(jq -c --arg v "$v" --arg sha "$sha" '. + [{version: $v, sha: $sha}]' <<<"$releases")
	done
	[[ $releases != "[]" ]] || echo "no unreleased CHANGELOG.md section since $last"
	out releases "$releases"
}

tag() {
	local version=$1 commit=$2
	[[ $version =~ $semver ]] || die "invalid version '$version'"
	[[ $commit =~ ^[0-9a-f]{40}$ ]] || die "invalid commit '$commit'"
	if tag_exists "$version"; then
		local existing
		existing=$(git rev-parse "v$version^{commit}")
		[[ $existing == "$commit" ]] || die "v$version already exists at ${existing:0:7}, not ${commit:0:7}"
		echo "tag v$version already points at ${commit:0:7}"
		return
	fi
	git tag "v$version" "$commit"
	git push -q origin "v$version"
	echo "pushed tag v$version at ${commit:0:7}"
}

[[ $# -ge 1 ]] || usage
case "$1" in
update) update ;;
check) check ;;
release-plan) release_plan ;;
tag)
	[[ $# -eq 3 ]] || usage
	tag "${2#v}" "$3"
	;;
notes)
	[[ $# -eq 3 || $# -eq 4 ]] || usage
	notes "$2" "$3" "${4:-}"
	;;
extract)
	[[ $# -eq 2 ]] || usage
	extract "${2#v}" || die "CHANGELOG.md has no section for ${2#v}"
	;;
release-notes)
	[[ $# -eq 2 ]] || usage
	release_notes "$2"
	;;
*) usage ;;
esac
