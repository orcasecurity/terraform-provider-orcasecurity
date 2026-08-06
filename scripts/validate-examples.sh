#!/usr/bin/env bash
# terraform validate shift-left example fragments (tfplugindocs embeds examples/ into docs/).
# Fragments get a temp provider block; SKIP lists dirs with cross-directory refs.
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

  # Only shift-left examples; others need cross-directory refs.
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

validate_snippet() {
  local case_dir="$1" name="$2"
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
    echo "ok       $name"
  else
    echo "INVALID  $name"
    echo "$output" | sed 's/^/         /'
    failed+=("$name")
  fi
}

# shift_left docs also hand-write ```terraform snippets straight in templates/*.tmpl,
# which never pass through examples/, so validate those here too.
for tmpl in $(find templates -iname '*shift_left*.md.tmpl' | sort); do
  block_num=0
  in_block=0
  block_file=""
  while IFS='' read -r line; do
    if [[ "$in_block" -eq 0 && "$line" == '```terraform' ]]; then
      in_block=1
      block_file="$work/snippet.tf"
      : >"$block_file"
      continue
    fi
    if [[ "$in_block" -eq 1 && "$line" == '```' ]]; then
      in_block=0
      case_dir="$work/case"
      rm -rf "$case_dir"
      mkdir -p "$case_dir"
      cp "$block_file" "$case_dir/snippet.tf"
      validate_snippet "$case_dir" "$tmpl block $block_num"
      block_num=$((block_num + 1))
      continue
    fi
    if [[ "$in_block" -eq 1 ]]; then
      echo "$line" >>"$block_file"
    fi
  done <"$tmpl"
done

echo
if ((${#failed[@]} > 0)); then
  echo "$checked example(s) checked, ${#failed[@]} invalid:"
  printf '  %s\n' "${failed[@]}"
  exit 1
fi
echo "$checked example(s) checked, all valid."
