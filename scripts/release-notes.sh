#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'EOF'
usage:
  scripts/release-notes.sh notes <version>             print the generated notes for <version>
  scripts/release-notes.sh generate <version> [date]   add a generated <version> section to the top of CHANGELOG.md
  scripts/release-notes.sh extract <version>           print the <version> section body from CHANGELOG.md

env:
  BASE_REF  commit to generate from when <version> is not tagged (default: HEAD)
  ENTRY     extra commit subject appended on top of BASE_REF, e.g. "feat(x): y (#280)"
  PREVIOUS_TAG, PREVIOUS_REF  start the range at PREVIOUS_REF (a merged release not tagged yet)
EOF
	exit 2
}

die() {
	echo "release-notes: $*" >&2
	exit 1
}

repo_root=$(git rev-parse --show-toplevel)
changelog="$repo_root/CHANGELOG.md"

extract() {
	local version=$1
	awk -v v="$version" '
		index($0, "## " v " ") == 1 || $0 == "## " v { found = 1; next }
		found && /^## / { exit }
		found { print }
		END { if (!found) exit 1 }
	' "$changelog" | perl -0777 -pe 's/\A\s+//; s/\s+\z/\n/'
}

repo_url() {
	local url
	url=$(git -C "$repo_root" remote get-url origin)
	url=${url%.git}
	url=${url/#git@github.com:/https://github.com/}
	echo "$url"
}

changelog_config() {
	awk '
		/^changelog:/ { inblock = 1; print; next }
		inblock && /^[^[:space:]#]/ { exit }
		inblock { print }
	' "$repo_root/.goreleaser.yml"
}

render() {
	local url=$1
	REPO_URL=$url perl -ne '
		my $u = $ENV{REPO_URL};
		next if /^## Changelog$/ || /^\s*$/;
		if (/^### /) { print "\n" if $seen++; print "$_\n"; next }
		if (/^\* ([0-9a-f]{7,40}) (.*)$/) {
			my ($sha, $msg) = ($1, $2);
			if ($msg =~ /\(#\d+\)/) {
				$msg =~ s{\(#(\d+)\)}{([#$1]($u/pull/$1))}g;
			} else {
				$msg .= " ([$sha]($u/commit/$sha))";
			}
			$_ = "* $msg\n";
		}
		print;
	'
}

notes() {
	local version=$1
	local tag="v$version" goreleaser=${GORELEASER:-goreleaser} base=${BASE_REF:-HEAD} entry=${ENTRY:-}
	command -v "$goreleaser" >/dev/null || die "goreleaser not found; run via 'make changelog'"

	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT

	# GoReleaser needs a real tag to compute the range, so an unreleased version
	# (and the PR's own squash commit) exists only inside this throwaway clone.
	git clone -q --shared --no-checkout "$repo_root" "$tmp/src"
	if [[ -z $entry ]] && git -C "$repo_root" rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
		git -C "$tmp/src" checkout -q --detach "$tag"
	else
		git -C "$tmp/src" checkout -q --detach "$(git -C "$repo_root" rev-parse "$base^{commit}")"
		if [[ -n $entry ]]; then
			git -C "$tmp/src" -c user.name=changelog -c user.email=changelog@localhost \
				commit -q --allow-empty --no-verify -m "$entry"
		fi
		git -C "$tmp/src" tag -f "$tag" >/dev/null
	fi
	if [[ -n ${PREVIOUS_TAG:-} && -n ${PREVIOUS_REF:-} ]]; then
		git -C "$tmp/src" tag -f "$PREVIOUS_TAG" "$(git -C "$repo_root" rev-parse "$PREVIOUS_REF^{commit}")" >/dev/null
	fi

	{
		echo "project_name: $(basename "$(repo_url)")"
		echo "dist: $tmp/dist"
		echo "builds:"
		echo "    - skip: true"
		changelog_config
	} >"$tmp/goreleaser.yml"

	(cd "$tmp/src" && GITHUB_TOKEN=${GITHUB_TOKEN:-unused} "$goreleaser" release \
		--clean --skip=publish,validate --config "$tmp/goreleaser.yml" >"$tmp/goreleaser.log" 2>&1) ||
		{ cat "$tmp/goreleaser.log" >&2; die "goreleaser failed"; }

	echo "release-notes: $version $(grep -o 'previous_tag=[^ ]*' "$tmp/goreleaser.log" | head -1)" >&2
	local body
	body=$(render "$(repo_url)" <"$tmp/dist/CHANGELOG.md")
	echo "${body:-No user-facing changes.}"
}

generate() {
	local version=$1 date=${2:-$(date '+%B %-d, %Y')}
	[[ -f $changelog ]] || die "CHANGELOG.md not found"
	if extract "$version" >/dev/null 2>&1; then
		die "CHANGELOG.md already has a section for $version"
	fi

	local body section out
	body=$(notes "$version")
	section=$(mktemp)
	out=$(mktemp)
	printf '## %s (%s)\n\n%s\n\n' "$version" "$date" "$body" >"$section"
	awk -v section="$section" '
		!done && /^## / { while ((getline line < section) > 0) print line; done = 1 }
		{ print }
		END { if (!done) while ((getline line < section) > 0) print line }
	' "$changelog" >"$out"
	cp "$out" "$changelog"
	rm -f "$section" "$out"
	echo "CHANGELOG.md: added $version"
}

[[ $# -ge 2 ]] || usage
cmd=$1 version=${2#v}
case "$cmd" in
notes) notes "$version" ;;
generate) generate "$version" "${3:-}" ;;
extract) extract "$version" || die "CHANGELOG.md has no section for $version" ;;
*) usage ;;
esac
