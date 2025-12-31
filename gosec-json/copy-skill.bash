#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE' >&2
Usage: copy-skill.bash <destination-dir>

Copies the skill-gosec-json-extract template into <destination-dir>
and installs the gosec-json-extract.bash script.
Example:
  copy-skill.bash .
USAGE
}

if [[ $# -ne 1 ]]; then
  usage
  exit 2
fi

dest_dir="$1"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
template_dir="$script_dir/skill-gosec-json-extract"
source_script="$script_dir/gosec-json-extract.bash"

if [[ ! -d "$template_dir" ]]; then
  echo "Template not found: $template_dir" >&2
  exit 1
fi
if [[ ! -f "$source_script" ]]; then
  echo "Source script not found: $source_script" >&2
  exit 1
fi
if [[ ! -d "$dest_dir" ]]; then
  echo "Destination is not a directory: $dest_dir" >&2
  exit 1
fi

dest_dir="$(cd "$dest_dir" && pwd -P)"
dest_skill_dir="$dest_dir/skill-gosec-json-extract"

if [[ -e "$dest_skill_dir" ]]; then
  echo "Destination already exists: $dest_skill_dir" >&2
  exit 1
fi

cp -R "$template_dir" "$dest_dir/"
mkdir -p "$dest_skill_dir/scripts"
cp "$source_script" "$dest_skill_dir/scripts/gosec-json-extract.bash"

chmod +x "$dest_skill_dir/scripts/gosec-json-extract.bash"

echo "Copied skill to: $dest_skill_dir"
