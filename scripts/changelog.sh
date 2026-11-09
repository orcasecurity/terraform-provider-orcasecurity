#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'EOF'
usage:
  scripts/changelog.sh update                  write this PR's section into CHANGELOG.md (run via 'make changelog')
  scripts/changelog.sh check                   CI: fail unless the PR adds the expected CHANGELOG.md section
  scripts/changelog.sh release-plan            CI: find a merged, untagged CHANGELOG.md section to release
  scripts/changelog.sh tag <version> <commit>  CI: tag <commit> as v<version> and push the tag

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

latest_tag() {
	local ref=$1
	git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname --merged "$ref" | sed -n 1p
}

# The newest version on the base branch: its latest tag, or a merged section
# that is not tagged yet because its release is still running.
base_version() {
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

# When the newest merged section is not tagged yet, its commit starts the range.
previous_release_env() {
	local ref=$1 version=$2 tag
	tag=$(latest_tag "$ref")
	if [[ $version != "${tag#v}" ]]; then
		PREVIOUS_TAG="v$version"
		PREVIOUS_REF=$(section_commit "$ref" "$version" "$tag")
		export PREVIOUS_TAG PREVIOUS_REF
	fi
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

# Only "## X.Y.Z (Month D, YYYY)" headings are releases; this skips the old
# "## 0.1.0 (Unreleased)" stub and "## 0.0.54 and earlier".
versions_in() {
	sed -nE 's/^## ([0-9]+\.[0-9]+\.[0-9]+) \([A-Z][a-z]+ [0-9]{1,2}, [0-9]{4}\)$/\1/p'
}

section_versions() {
	local file=$1
	versions_in <"$file"
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
	for v in $(section_versions "$head_file"); do
		if ! tag_exists "$v" && ! grep -qx "$v" <(section_versions "$base_file"); then
			echo "$v"
		fi
	done
}

section_body() {
	local file=$1 version=$2
	awk -v v="$version" '
		index($0, "## " v " ") == 1 { found = 1; next }
		found && /^## / { exit }
		found { print }
	' "$file"
}

backfill_missing_releases() {
	local base=$1 newest t v
	newest=$(section_versions CHANGELOG.md | sort -V | tail -1)
	[[ -n $newest ]] || return 0
	for t in $(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=v:refname --merged "$base"); do
		v=${t#v}
		[[ $v != "$newest" && $(printf '%s\n%s\n' "$newest" "$v" | sort -V | tail -1) == "$v" ]] || continue
		BASE_REF="" ENTRY="" scripts/release-notes.sh generate "$v" "$(git log -1 --format=%cd --date=format:'%B %-d, %Y' "$t")"
	done
}

update() {
	local pr number title labels base last version
	local -a selector=()
	command -v gh >/dev/null || die "gh CLI is required to read the PR title and labels"
	[[ -z ${PR_NUMBER:-} ]] || selector=("$PR_NUMBER")
	pr=$(gh pr view ${selector[@]+"${selector[@]}"} --json number,title,labels --jq '[.number, .title, ([.labels[].name] | join(","))] | @tsv') ||
		die "no PR found for this branch; open it first (a draft is fine), then rerun 'make changelog'"
	IFS=$'\t' read -r number title labels <<<"$pr"

	git fetch -q origin "$base_branch" --tags
	base=origin/$base_branch
	local base_file v
	if ! git diff --quiet "$(git merge-base HEAD "$base")" "$base" -- CHANGELOG.md; then
		die "CHANGELOG.md changed on $base_branch since this branch started (another PR was released); merge or rebase on $base_branch, then rerun 'make changelog'"
	fi
	base_file=$(mktemp)
	git show "$base:CHANGELOG.md" >"$base_file"
	for v in $(section_versions CHANGELOG.md); do
		if ! tag_exists "$v" && ! grep -qx "$v" <(section_versions "$base_file"); then
			remove_section CHANGELOG.md "$v"
		fi
	done
	rm -f "$base_file"
	backfill_missing_releases "$base"

	if has_label "$labels" no-release; then
		echo "PR #$number has the no-release label: no section added (this change ships with the next release)"
		return
	fi

	last=$(base_version "$base")
	[[ -n $last ]] || die "no v* tag found on $base"
	version=$(next_version "$last" "$labels")
	previous_release_env "$base" "$last"
	BASE_REF=$base ENTRY="$title (#$number)" scripts/release-notes.sh generate "$version"
	echo "PR #$number releases v$version on merge. Review the section, then commit CHANGELOG.md."
}

check() {
	local number=${PR_NUMBER:?} title=${PR_TITLE:?} labels=${PR_LABELS:-} author=${PR_AUTHOR:-}
	local base=${BASE_REF:-HEAD^1} base_file head_file last expected version body missing ref
	local -a new
	[[ $number =~ ^[0-9]+$ ]] || die "invalid PR number '$number'"

	if [[ $author == "dependabot[bot]" ]]; then
		summary "✅ Dependabot PR: exempt. Its bump is listed in the next release's section."
		return
	fi

	base_file=$(mktemp)
	head_file=$(mktemp)
	git show "$base:CHANGELOG.md" >"$base_file"
	cp CHANGELOG.md "$head_file"
	mapfile -t new < <(untagged_new_versions "$head_file" "$base_file")

	if has_label "$labels" no-release; then
		((${#new[@]} == 0)) || fail "This PR has the \`no-release\` label but adds a section for ${new[*]}. Remove the section or the label."
		summary "✅ \`no-release\` label: no CHANGELOG.md section needed. This change ships with the next release."
		return
	fi

	last=$(base_version "$base")
	[[ -n $last ]] || die "no v* tag found on $base"
	expected=$(next_version "$last" "$labels")
	previous_release_env "$base" "$last"
	((${#new[@]} > 0)) || fail "This PR does not add a CHANGELOG.md section. Merging it releases **v$expected**, so it needs a \`## $expected (...)\` section."
	((${#new[@]} == 1)) || fail "This PR adds more than one new section (${new[*]}); it should add only \`## $expected\`."
	version=${new[0]}
	[[ $version == "$expected" ]] || fail "The new section is \`## $version\`, but merging this PR releases **v$expected** (latest version on $base_branch: $last, labels: ${labels:-none})."

	body=$(section_body "$head_file" "$version")
	[[ $body == *"#$number"* ]] || fail "The \`## $version\` section has no entry for this PR (#$number)."

	missing=""
	for ref in $(BASE_REF=$base ENTRY="$title (#$number)" scripts/release-notes.sh notes "$version" | grep -oE '\[#[0-9]+\]' | tr -d '[]' | sort -u); do
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

# Every merged section without a tag is released, oldest first: when several
# PRs merge before a run starts, GitHub replaces the queued runs with one.
# Each version must be exactly what its PR's labels allow, as the PR check
# requires, because that check is not a required status check.
release_plan() {
	local last v sha previous labels expected releases="[]"
	last=$(latest_tag HEAD)
	[[ -n $last ]] || die "no v* tag found"
	previous=${last#v}
	for v in $(section_versions CHANGELOG.md | sort -V); do
		tag_exists "$v" && continue
		[[ $v != "${last#v}" && $(printf '%s\n%s\n' "${last#v}" "$v" | sort -V | tail -1) == "$v" ]] || continue
		sha=$(section_commit HEAD "$v" "$last")
		[[ -n $sha ]] || die "cannot find the commit that added the $v section"
		labels=$(merged_pr_labels "$sha") || die "cannot read the PR labels for ${sha:0:7}; nothing was released"
		expected=$(next_version "$previous" "$labels")
		[[ $v == "$expected" ]] ||
			die "CHANGELOG.md section $v (${sha:0:7}) should be $expected after $previous (PR labels: ${labels:-none}); nothing was released"
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
*) usage ;;
esac
