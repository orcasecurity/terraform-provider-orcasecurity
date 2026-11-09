#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'EOF'
usage:
  scripts/release-pr.sh plan                    decide what master needs: release_version/release_sha and/or prepare_version
  scripts/release-pr.sh prepare <version>       open or refresh the "Release v<version>" PR from release/next
  scripts/release-pr.sh tag <version> <commit>  tag <commit> as v<version> and push the tag
EOF
	exit 2
}

die() {
	echo "release-pr: $*" >&2
	exit 1
}

branch=release/next
semver='^[0-9]+\.[0-9]+\.[0-9]+$'
release_subject='^Release v([0-9]+\.[0-9]+\.[0-9]+)( \(#[0-9]+\))?$'

out() {
	local key=$1 value=$2
	echo "$key=$value" >>"${GITHUB_OUTPUT:-/dev/stdout}"
}

tag_exists() {
	local version=$1
	git rev-parse -q --verify "refs/tags/v$version" >/dev/null
}

latest_tag() {
	git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname --merged HEAD | head -1
}

next_patch() {
	local v=${1#v}
	echo "${v%.*}.$((${v##*.} + 1))"
}

# A merged Release PR is found anywhere since the last tag, not only at HEAD:
# a queued run for the merge can be replaced by a later push to master.
unreleased_release_commit() {
	local since=$1 sha subject
	while read -r sha subject; do
		if [[ $subject =~ $release_subject ]] && ! tag_exists "${BASH_REMATCH[1]}"; then
			echo "$sha ${BASH_REMATCH[1]}"
			return
		fi
	done < <(git log --format='%H %s' "$since..HEAD")
}

open_pr_version() {
	local title
	title=$(gh pr list --head "$branch" --state open --json title --jq '.[0].title // ""')
	if [[ $title =~ ^Release\ v([0-9]+\.[0-9]+\.[0-9]+)$ ]] && ! tag_exists "${BASH_REMATCH[1]}"; then
		echo "${BASH_REMATCH[1]}"
	fi
}

plan() {
	local last release_sha="" release_version="" base base_rev version=""
	last=$(latest_tag)
	[[ -n $last ]] || die "no v* tag found"

	read -r release_sha release_version <<<"$(unreleased_release_commit "$last")" || true
	if [[ -n $release_version ]]; then
		[[ $'\n'$(git show "$release_sha:CHANGELOG.md") == *$'\n'"## $release_version "* ]] ||
			die "$(git log -1 --format=%s "$release_sha") has no CHANGELOG.md section for $release_version"
		echo "release v$release_version at ${release_sha:0:7}"
		out release_version "$release_version"
		out release_sha "$release_sha"
		base="v$release_version"
		base_rev=$release_sha
	else
		base=$last
		base_rev=$last
	fi

	if [[ -z $(git rev-list "$base_rev..HEAD") ]]; then
		echo "nothing to prepare since $base"
		return
	fi

	version=${REQUESTED_VERSION:-}
	[[ -n $version ]] || version=$(open_pr_version)
	[[ $version != "$release_version" ]] || version=""
	[[ -n $version ]] || version=$(next_patch "$base")
	version=${version#v}
	[[ $version =~ $semver ]] || die "invalid version '$version'"
	if tag_exists "$version" || [[ $version == "$release_version" ]]; then
		die "v$version is already released"
	fi

	echo "prepare v$version (commits since $base)"
	out prepare_version "$version"
}

backfill_missing_releases() {
	local newest t v
	newest=$(sed -n 's/^## \([0-9][0-9.]*\) .*/\1/p' CHANGELOG.md | sort -V | tail -1)
	[[ -n $newest ]] || return 0
	for t in $(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=v:refname --merged HEAD); do
		v=${t#v}
		[[ $v != "$newest" && $(printf '%s\n%s\n' "$newest" "$v" | sort -V | tail -1) == "$v" ]] || continue
		scripts/release-notes.sh generate "$v" "$(git log -1 --format=%cd --date=format:'%B %-d, %Y' "$t")"
	done
}

prepare() {
	local version=$1 number body
	[[ $version =~ $semver ]] || die "invalid version '$version'"

	git switch -q -C "$branch"
	backfill_missing_releases
	scripts/release-notes.sh generate "$version"
	git add CHANGELOG.md
	git -c user.name="github-actions[bot]" \
		-c user.email="41898282+github-actions[bot]@users.noreply.github.com" \
		commit -q -m "Release v$version"
	git push -q --force origin "$branch"

	body=$(mktemp)
	{
		echo "Merging this PR tags **v$version** and publishes the release with the notes below."
		echo "The Release PR workflow rebuilds this PR on every push to master. To reword an entry, edit CHANGELOG.md on this branch right before merging; edits are replaced if master changes first."
		echo
		scripts/release-notes.sh extract "$version"
	} >"$body"

	number=$(gh pr list --head "$branch" --state open --json number --jq '.[0].number // empty')
	if [[ -n $number ]]; then
		gh pr edit "$number" --title "Release v$version" --body-file "$body"
		echo "updated PR #$number: Release v$version"
	else
		gh pr create --base master --head "$branch" --title "Release v$version" --body-file "$body"
		echo "opened PR: Release v$version"
	fi
	rm -f "$body"
}

tag() {
	local version=$1 commit=$2
	[[ $version =~ $semver ]] || die "invalid version '$version'"
	[[ $commit =~ ^[0-9a-f]{40}$ ]] || die "invalid commit '$commit'"
	if tag_exists "$version"; then
		die "v$version already exists"
	fi
	git tag "v$version" "$commit"
	git push -q origin "v$version"
	echo "pushed tag v$version at ${commit:0:7}"
}

[[ $# -ge 1 ]] || usage
case "$1" in
plan) plan ;;
prepare)
	[[ $# -eq 2 ]] || usage
	prepare "${2#v}"
	;;
tag)
	[[ $# -eq 3 ]] || usage
	tag "${2#v}" "$3"
	;;
*) usage ;;
esac
