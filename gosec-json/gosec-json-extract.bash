#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE' >&2
Usage: gosec-json-extract.bash [options] [path]

Extract findings from a gosec JSON report and print CLI-friendly output.

Options:
  --format <template>       Output template (default shown below)
  --group <field>           Group by: severity|confidence|cwe|rule_id|file
  --sort <field>            Sort by: severity|confidence|cwe|rule_id|file|line
  --severity <value>        Filter by severity (repeatable)
  --confidence <value>      Filter by confidence (repeatable)
  --rule-id <value>         Filter by rule_id (repeatable)
  --cwe <value>             Filter by CWE id (repeatable, accepts CWE-###)
  --file <regex>            Filter by file path regex
  --stats                   Print stats only and exit
  -h, --help                Show this help

Template placeholders:
  {severity} {confidence} {rule_id} {cwe} {cwe_id} {cwe_url}
  {file} {line} {column} {location} {details} {code}

Default template:
  {severity} {confidence} {rule_id} {cwe} {location} {details}

Examples:
  gosec-json-extract.bash report.json
  gosec-json-extract.bash --severity HIGH --sort file report.json
  gosec-json-extract.bash --group rule_id --format "{rule_id} {location} {details}" report.json
  cat report.json | gosec-json-extract.bash --confidence MEDIUM
USAGE
}

if ! command -v jq >/dev/null 2>&1; then
  echo "jq(1) is required but not found in PATH" >&2
  exit 1
fi

fmt="{severity} {confidence} {rule_id} {cwe} {location} {details}"
group_field=""
sort_field=""
file_re=""
stats_only=0
input=""

severities=()
confidences=()
rules=()
cwes=()

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
    --severity)
      severities+=("$2")
      shift 2
      ;;
    --confidence)
      confidences+=("$2")
      shift 2
      ;;
    --rule-id|--rule)
      rules+=("$2")
      shift 2
      ;;
    --cwe)
      cwes+=("${2#CWE-}")
      shift 2
      ;;
    --file)
      file_re="$2"
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

if [[ ${#severities[@]} -eq 0 ]]; then
  severities_json='[]'
else
  severities_json=$(printf '%s\n' "${severities[@]}" | jq -R . | jq -s .)
fi
if [[ ${#confidences[@]} -eq 0 ]]; then
  confidences_json='[]'
else
  confidences_json=$(printf '%s\n' "${confidences[@]}" | jq -R . | jq -s .)
fi
if [[ ${#rules[@]} -eq 0 ]]; then
  rules_json='[]'
else
  rules_json=$(printf '%s\n' "${rules[@]}" | jq -R . | jq -s .)
fi
if [[ ${#cwes[@]} -eq 0 ]]; then
  cwes_json='[]'
else
  cwes_json=$(printf '%s\n' "${cwes[@]}" | jq -R . | jq -s .)
fi

if [[ $stats_only -eq 1 ]]; then
  jq -r '
    "GosecVersion: \(.GosecVersion // "")",
    "Issues: \((.Issues // []) | length)",
    "Files: \(.Stats.files // 0)",
    "Lines: \(.Stats.lines // 0)",
    "Nosec: \(.Stats.nosec // 0)",
    "Found: \(.Stats.found // 0)"
  ' "$input"
  exit 0
fi

jq -r \
  --arg fmt "$fmt" \
  --arg group "$group_field" \
  --arg sort "$sort_field" \
  --arg file_re "$file_re" \
  --argjson severities "$severities_json" \
  --argjson confidences "$confidences_json" \
  --argjson rules "$rules_json" \
  --argjson cwes "$cwes_json" \
  '
  def severity_rank:
    if . == "HIGH" then 3
    elif . == "MEDIUM" then 2
    elif . == "LOW" then 1
    else 0 end;
  def confidence_rank:
    if . == "HIGH" then 3
    elif . == "MEDIUM" then 2
    elif . == "LOW" then 1
    else 0 end;

  def replace($needle; $value):
    split($needle) | join($value);

  def format_line($template):
    . as $issue
    | $template
    | replace("{severity}"; ($issue.severity // ""))
    | replace("{confidence}"; ($issue.confidence // ""))
    | replace("{rule_id}"; ($issue.rule_id // ""))
    | replace("{cwe_id}"; ($issue.cwe.id // ""))
    | replace("{cwe_url}"; ($issue.cwe.url // ""))
    | replace("{cwe}"; (if $issue.cwe.id then ("CWE-" + ($issue.cwe.id | tostring)) else "" end))
    | replace("{file}"; ($issue.file // ""))
    | replace("{line}"; (($issue.line // "") | tostring))
    | replace("{column}"; (($issue.column // "") | tostring))
    | replace("{location}"; (if $issue.file then ($issue.file + ":" + (($issue.line // "") | tostring) + ":" + (($issue.column // "") | tostring)) else "" end))
    | replace("{details}"; ($issue.details // ""))
    | replace("{code}"; ($issue.code // ""));

  def key_for($field):
    if $field == "severity" then (.severity // "")
    elif $field == "confidence" then (.confidence // "")
    elif $field == "cwe" then (.cwe.id // "")
    elif $field == "rule_id" then (.rule_id // "")
    elif $field == "file" then (.file // "")
    elif $field == "line" then ((.line // 0) | tostring)
    else "" end;

  def group_sort_key($field):
    if $field == "severity" then (-(.severity | severity_rank))
    elif $field == "confidence" then (-(.confidence | confidence_rank))
    elif $field == "line" then (.line // 0)
    else key_for($field) end;

  def sort_key($field):
    if $field == "severity" then [-(.severity | severity_rank), (.file // ""), (.line // 0)]
    elif $field == "confidence" then [-(.confidence | confidence_rank), (.file // ""), (.line // 0)]
    elif $field == "line" then (.line // 0)
    else key_for($field) end;

  def apply_filters:
    . as $issue
    | select(($severities | length == 0) or ($severities | index($issue.severity)))
    | select(($confidences | length == 0) or ($confidences | index($issue.confidence)))
    | select(($rules | length == 0) or ($rules | index($issue.rule_id)))
    | select(($cwes | length == 0) or ($cwes | index($issue.cwe.id)))
    | select(($file_re == "") or ($issue.file | test($file_re)));

  .Issues // []
  | map(apply_filters)
  | if $group != "" then
      (sort_by(group_sort_key($group))
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
