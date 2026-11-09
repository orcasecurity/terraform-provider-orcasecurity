#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'EOF'
usage:
  scripts/release-notes.sh generate <version> [date]   add a <version> section to the top of CHANGELOG.md
  scripts/release-notes.sh extract <version>           print the <version> section body (for the GitHub release)
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

generate() {
	local version=$1 date=${2:-$(date '+%B %-d, %Y')}
	local tag="v$version"
	local goreleaser=${GORELEASER:-goreleaser}

	command -v "$goreleaser" >/dev/null || die "goreleaser not found; run via 'make release-notes VERSION=$version'"
	[[ -f $changelog ]] || die "CHANGELOG.md not found"
	if extract "$version" >/dev/null 2>&1; then
		die "CHANGELOG.md already has a section for $version"
	fi

	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT

	# The tag must exist for GoReleaser to compute the range, so an unreleased
	# version is tagged only inside this throwaway clone, never in the real repo.
	git clone -q --shared --no-checkout "$repo_root" "$tmp/src"
	if git -C "$repo_root" rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
		git -C "$tmp/src" checkout -q --detach "$tag"
	else
		git -C "$tmp/src" checkout -q --detach "$(git -C "$repo_root" rev-parse HEAD)"
		git -C "$tmp/src" tag "$tag"
	fi

	{
		echo "project_name: $(basename "$(repo_url)")"
		echo "dist: $tmp/dist"
		echo "builds:"
		echo "    - skip: true"
		changelog_config
	} >"$tmp/goreleaser.yml"

	(cd "$tmp/src" && GITHUB_TOKEN=${GITHUB_TOKEN:-unused} "$goreleaser" release \
		--clean --skip=publish --config "$tmp/goreleaser.yml" >"$tmp/goreleaser.log" 2>&1) ||
		{ cat "$tmp/goreleaser.log" >&2; die "goreleaser failed"; }

	local body
	body=$(render "$(repo_url)" <"$tmp/dist/CHANGELOG.md")
	[[ -n $body ]] || body="No user-facing changes."

	local section="$tmp/section.md"
	printf '## %s (%s)\n\n%s\n\n' "$version" "$date" "$body" >"$section"

	awk -v section="$section" '
		!done && /^## / { while ((getline line < section) > 0) print line; done = 1 }
		{ print }
		END { if (!done) while ((getline line < section) > 0) print line }
	' "$changelog" >"$tmp/CHANGELOG.md"
	cp "$tmp/CHANGELOG.md" "$changelog"

	echo "CHANGELOG.md: added $version ($(grep -o 'previous_tag=[^ ]*' "$tmp/goreleaser.log" | head -1))"
}

[[ $# -ge 2 ]] || usage
cmd=$1 version=${2#v}
case "$cmd" in
generate) generate "$version" "${3:-}" ;;
extract) extract "$version" || die "CHANGELOG.md has no section for $version" ;;
*) usage ;;
esac
