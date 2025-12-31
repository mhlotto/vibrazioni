#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE' >&2
Usage: govulncheck-json-extract.bash [options] [path]

Extract findings from govulncheck JSON output and print CLI-friendly output.

Options:
  --format <template>       Output template (default shown below)
  --group <field>           Group by: osv|module|package|fixed_version
  --sort <field>            Sort by: osv|module|package|fixed_version|summary
  --osv <id>                Filter by OSV id (repeatable)
  --module <path>           Filter by module path (repeatable)
  --package <path>          Filter by package path (repeatable)
  --fixed-version <ver>     Filter by fixed version (repeatable)
  --alias <id>              Filter by alias (repeatable, e.g. CVE-XXXX-YYYY)
  --stats                   Print stats only and exit
  -h, --help                Show this help

Template placeholders:
  {osv} {osv_id} {summary} {details} {aliases}
  {module} {module_version} {package} {fixed_version}
  {trace} {url}

Default template:
  {osv} {module}@{module_version} fixed:{fixed_version} {summary}

Examples:
  govulncheck-json-extract.bash report.json
  govulncheck-json-extract.bash --module golang.org/x/crypto report.json
  govulncheck-json-extract.bash --group module --format "{module} {osv} {summary}" report.json
  cat report.json | govulncheck-json-extract.bash --stats
USAGE
}

if ! command -v jq >/dev/null 2>&1; then
  echo "jq(1) is required but not found in PATH" >&2
  exit 1
fi

fmt="{osv} {module}@{module_version} fixed:{fixed_version} {summary}"
group_field=""
sort_field=""
stats_only=0
input=""

osv_ids=()
modules=()
packages=()
fixed_versions=()
aliases=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --format)
      fmt="$2"
      shift 2
      ;;
    --group)
      group_field="$2"
      shift 2
      ;;
    --sort)
      sort_field="$2"
      shift 2
      ;;
    --osv)
      osv_ids+=("$2")
      shift 2
      ;;
    --module)
      modules+=("$2")
      shift 2
      ;;
    --package)
      packages+=("$2")
      shift 2
      ;;
    --fixed-version)
      fixed_versions+=("$2")
      shift 2
      ;;
    --alias)
      aliases+=("$2")
      shift 2
      ;;
    --stats)
      stats_only=1
      shift
      ;;
    --)
      shift
      break
      ;;
    -* )
      echo "Unknown option: $1" >&2
      usage
      exit 2
      ;;
    * )
      if [[ -n "$input" ]]; then
        echo "Unexpected argument: $1" >&2
        usage
        exit 2
      fi
      input="$1"
      shift
      ;;
  esac
 done

if [[ -z "$input" ]]; then
  if [[ -t 0 ]]; then
    echo "Missing input path (or pipe JSON on stdin)." >&2
    usage
    exit 2
  fi
  input="-"
fi

if [[ ${#osv_ids[@]} -eq 0 ]]; then
  osv_json='[]'
else
  osv_json=$(printf '%s\n' "${osv_ids[@]}" | jq -R . | jq -s .)
fi
if [[ ${#modules[@]} -eq 0 ]]; then
  modules_json='[]'
else
  modules_json=$(printf '%s\n' "${modules[@]}" | jq -R . | jq -s .)
fi
if [[ ${#packages[@]} -eq 0 ]]; then
  packages_json='[]'
else
  packages_json=$(printf '%s\n' "${packages[@]}" | jq -R . | jq -s .)
fi
if [[ ${#fixed_versions[@]} -eq 0 ]]; then
  fixed_versions_json='[]'
else
  fixed_versions_json=$(printf '%s\n' "${fixed_versions[@]}" | jq -R . | jq -s .)
fi
if [[ ${#aliases[@]} -eq 0 ]]; then
  aliases_json='[]'
else
  aliases_json=$(printf '%s\n' "${aliases[@]}" | jq -R . | jq -s .)
fi

if [[ $stats_only -eq 1 ]]; then
  jq -s -r '
    . as $docs
    | ($docs | map(select(has("config")) | .config) | first // {}) as $config
    | ($docs | map(select(has("SBOM")) | .SBOM) | first // {}) as $sbom
    | ($docs | map(select(has("finding")) | .finding)) as $findings
    | ($docs | map(select(has("osv")) | .osv)) as $osvs
    | "Scanner: \($config.scanner_name // "") \($config.scanner_version // "")"
    , "DB: \($config.db // "") (last modified: \($config.db_last_modified // ""))"
    , "Go: \($config.go_version // "") scan_level: \($config.scan_level // "") scan_mode: \($config.scan_mode // "")"
    , "SBOM modules: \(($sbom.modules // []) | length)"
    , "OSV entries: \($osvs | length)"
    , "Findings: \($findings | length)"
    , "Unique OSV findings: \($findings | map(.osv) | unique | length)"
    , "Unique modules in findings: \($findings | map(.trace[0].module) | unique | length)"
  ' "$input"
  exit 0
fi

jq -s -r \
  --arg fmt "$fmt" \
  --arg group "$group_field" \
  --arg sort "$sort_field" \
  --argjson osv_ids "$osv_json" \
  --argjson modules "$modules_json" \
  --argjson packages "$packages_json" \
  --argjson fixed_versions "$fixed_versions_json" \
  --argjson aliases "$aliases_json" \
  '
  def replace($needle; $value):
    split($needle) | join($value);

  def trace_string($trace):
    ($trace // [])
    | map(
        (.module // "")
        + (if .version then "@" + .version else "" end)
        + (if .package then "#" + .package else "" end)
        + (if .symbol then "#" + .symbol else "" end)
      )
    | join(" -> ");

  def enrich($osv_map):
    . as $f
    | ($osv_map[$f.osv] // {}) as $osv
    | {
        osv_id: ($f.osv // ""),
        fixed_version: ($f.fixed_version // ""),
        trace: ($f.trace // []),
        module: ($f.trace[0].module // ""),
        module_version: ($f.trace[0].version // ""),
        package: ($f.trace[0].package // ""),
        summary: ($osv.summary // ""),
        details: ($osv.details // ""),
        aliases: ($osv.aliases // []),
        url: ($osv.references[0].url // ""),
        trace_str: trace_string($f.trace)
      };

  def format_line($template):
    . as $i
    | $template
    | replace("{osv}"; ($i.osv_id // ""))
    | replace("{osv_id}"; ($i.osv_id // ""))
    | replace("{summary}"; ($i.summary // ""))
    | replace("{details}"; ($i.details // ""))
    | replace("{aliases}"; (if ($i.aliases | length) > 0 then ($i.aliases | join(",")) else "" end))
    | replace("{module}"; ($i.module // ""))
    | replace("{module_version}"; ($i.module_version // ""))
    | replace("{package}"; ($i.package // ""))
    | replace("{fixed_version}"; ($i.fixed_version // ""))
    | replace("{trace}"; ($i.trace_str // ""))
    | replace("{url}"; ($i.url // ""));

  def key_for($field):
    if $field == "osv" then (.osv_id // "")
    elif $field == "module" then (.module // "")
    elif $field == "package" then (.package // "")
    elif $field == "fixed_version" then (.fixed_version // "")
    elif $field == "summary" then (.summary // "")
    else "" end;

  def sort_key($field):
    if $field == "summary" then (.summary // "")
    else key_for($field) end;

  def apply_filters:
    . as $i
    | select(($osv_ids | length == 0) or ($osv_ids | index($i.osv_id)))
    | select(($modules | length == 0) or ($modules | index($i.module)))
    | select(($packages | length == 0) or ($packages | index($i.package)))
    | select(($fixed_versions | length == 0) or ($fixed_versions | index($i.fixed_version)))
    | select(($aliases | length == 0) or (any($i.aliases[]?; $aliases | index(.))));

  . as $docs
  | ($docs | map(select(has("osv")) | .osv) | map({(.id): .}) | add // {}) as $osv_map
  | ($docs | map(select(has("finding")) | .finding) | map(enrich($osv_map)))
  | map(apply_filters)
  | if $group != "" then
      (sort_by(key_for($group))
       | group_by(key_for($group))
       | map({key: (.[0] | key_for($group)), items: .}))
      | .[]
      | "== " + $group + ": " + (.key | tostring) + " ==",
        (.items
          | (if $sort != "" then sort_by(sort_key($sort)) else . end)
          | .[]
          | format_line($fmt)),
        ""
    else
      (if $sort != "" then sort_by(sort_key($sort)) else . end)
      | .[]
      | format_line($fmt)
    end
  ' "$input"
