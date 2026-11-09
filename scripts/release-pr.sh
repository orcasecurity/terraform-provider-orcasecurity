#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'EOF'
usage:
  scripts/release-pr.sh plan                decide what master needs: action=prepare|release|none, version=X.Y.Z
  scripts/release-pr.sh prepare <version>   open or refresh the "Release v<version>" PR from release/next
  scripts/release-pr.sh tag <version>       tag HEAD as v<version> and push the tag
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
	echo "$1=$2" >>"${GITHUB_OUTPUT:-/dev/stdout}"
}

tag_exists() {
	git rev-parse -q --verify "refs/tags/v$1" >/dev/null
}

latest_tag() {
	git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname --merged HEAD | head -1
}

next_patch() {
	local v=${1#v}
	echo "${v%.*}.$((${v##*.} + 1))"
}

open_pr_version() {
	local title
	title=$(gh pr list --head "$branch" --state open --json title --jq '.[0].title // ""')
	if [[ $title =~ ^Release\ v([0-9]+\.[0-9]+\.[0-9]+)$ ]] && ! tag_exists "${BASH_REMATCH[1]}"; then
		echo "${BASH_REMATCH[1]}"
	fi
}

plan() {
	local subject version last
	subject=$(git log -1 --format=%s HEAD)

	if [[ $subject =~ $release_subject ]] && ! tag_exists "${BASH_REMATCH[1]}"; then
		version=${BASH_REMATCH[1]}
		scripts/release-notes.sh extract "$version" >/dev/null ||
			die "'$subject' has no CHANGELOG.md section for $version"
		echo "release v$version"
		out action release
		out version "$version"
		return
	fi

	last=$(latest_tag)
	[ -n "$last" ] || die "no v* tag found"
	if [ -z "$(git rev-list "$last..HEAD")" ]; then
		echo "nothing to release since $last"
		out action none
		return
	fi

	version=${REQUESTED_VERSION:-}
	[ -n "$version" ] || version=$(open_pr_version)
	[ -n "$version" ] || version=$(next_patch "$last")
	version=${version#v}
	[[ $version =~ $semver ]] || die "invalid version '$version'"
	tag_exists "$version" && die "v$version is already released"

	echo "prepare v$version (commits since $last)"
	out action prepare
	out version "$version"
}

backfill_missing_releases() {
	local newest t v
	newest=$(sed -n 's/^## \([0-9][0-9.]*\) .*/\1/p' CHANGELOG.md | sort -V | tail -1)
	[ -n "$newest" ] || return 0
	for t in $(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=v:refname --merged HEAD); do
		v=${t#v}
		[ "$(printf '%s\n%s\n' "$newest" "$v" | sort -V | tail -1)" = "$v" ] && [ "$v" != "$newest" ] || continue
		scripts/release-notes.sh generate "$v" "$(git log -1 --format=%cd --date=format:'%B %-d, %Y' "$t")"
	done
}

prepare() {
	local version=$1 number
	[[ $version =~ $semver ]] || die "invalid version '$version'"

	git switch -q -C "$branch"
	backfill_missing_releases
	scripts/release-notes.sh generate "$version"
	git add CHANGELOG.md
	git -c user.name="github-actions[bot]" \
		-c user.email="41898282+github-actions[bot]@users.noreply.github.com" \
		commit -q -m "Release v$version"
	git push -q --force origin "$branch"

	local body
	body=$(mktemp)
	{
		echo "Merging this PR tags **v$version** and publishes the release with the notes below."
		echo "The Release PR workflow rebuilds this PR on every push to master. To reword an entry, edit CHANGELOG.md on this branch right before merging; edits are replaced if master changes first."
		echo
		scripts/release-notes.sh extract "$version"
	} >"$body"

	number=$(gh pr list --head "$branch" --state open --json number --jq '.[0].number // empty')
	if [ -n "$number" ]; then
		gh pr edit "$number" --title "Release v$version" --body-file "$body"
		echo "updated PR #$number: Release v$version"
	else
		gh pr create --base master --head "$branch" --title "Release v$version" --body-file "$body"
		echo "opened PR: Release v$version"
	fi
	rm -f "$body"
}

tag() {
	local version=$1
	[[ $version =~ $semver ]] || die "invalid version '$version'"
	tag_exists "$version" && die "v$version already exists"
	git tag "v$version" HEAD
	git push -q origin "v$version"
	echo "pushed tag v$version at $(git rev-parse --short HEAD)"
}

[ $# -ge 1 ] || usage
case "$1" in
plan) plan ;;
prepare) [ $# -eq 2 ] || usage; prepare "${2#v}" ;;
tag) [ $# -eq 2 ] || usage; tag "${2#v}" ;;
*) usage ;;
esac
