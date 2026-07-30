#!/usr/bin/env bash
#
# Validates the shift-left example configurations against the locally built provider.
#
# The examples under examples/ are the source that tfplugindocs embeds into docs/, so a
# stale attribute name there ships straight into the published documentation. Attribute
# names are not covered by any Go test, and `go generate` copies the files verbatim
# without checking them, so terraform validate is the only thing that catches a rename.
#
# Each example is a fragment (no terraform/provider block), so it is validated in a temp
# directory alongside a generated provider block. Fragments that reference variables or
# resources defined outside their own directory cannot be validated in isolation and are
# listed in SKIP below.
#
# Usage: scripts/validate-examples.sh
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

# Fragments that intentionally reference symbols declared outside their own directory.
SKIP=(
  examples
  examples/provider
)

work="$(mktemp -d)"
bin_dir="$work/bin"
trap 'rm -rf "$work"' EXIT

mkdir -p "$bin_dir"
go build -o "$bin_dir/terraform-provider-orcasecurity" .

cat >"$work/dev.tfrc" <<EOF
provider_installation {
  dev_overrides {
    "orcasecurity/orcasecurity" = "$bin_dir"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="$work/dev.tfrc"
export TF_IN_AUTOMATION=1

failed=()
checked=0

for dir in $(find examples -name '*.tf' -exec dirname {} \; | sort -u); do
  skip=false
  for skipped in "${SKIP[@]}"; do
    [[ "$dir" == "$skipped" ]] && skip=true && break
  done
  $skip && continue

  # Only shift-left examples are gated for now; the rest of examples/ predates this check
  # and relies on cross-directory references that cannot be validated in isolation.
  case "$dir" in
  *shift_left*) ;;
  *) continue ;;
  esac

  case_dir="$work/case"
  rm -rf "$case_dir"
  mkdir -p "$case_dir"
  cp "$dir"/*.tf "$case_dir/"
  cat >"$case_dir/zz_provider_for_validate.tf" <<'EOF'
terraform {
  required_providers {
    orcasecurity = { source = "orcasecurity/orcasecurity" }
  }
}
provider "orcasecurity" {
  api_endpoint = "https://api.orcasecurity.io"
  api_token    = "placeholder-for-validate-only"
}
EOF

  checked=$((checked + 1))
  if output=$(cd "$case_dir" && terraform validate -no-color 2>&1); then
    echo "ok       $dir"
  else
    echo "INVALID  $dir"
    echo "$output" | sed 's/^/         /'
    failed+=("$dir")
  fi
done

echo
if ((${#failed[@]} > 0)); then
  echo "$checked example(s) checked, ${#failed[@]} invalid:"
  printf '  %s\n' "${failed[@]}"
  exit 1
fi
echo "$checked example(s) checked, all valid."
