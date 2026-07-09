#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: repair-loop.sh PROJECT [MAX_PASSES]

Runs repeated mnemonic note-diagnostic passes. Actual repairs are delegated to
MNEMONIC_REPAIR_HOOK. The hook receives:

  1. The project selector.
  2. The path to the current JSON diagnostic report.

The script never selects repair candidates and never rebuilds the index.
Without a hook, it copies the report to the current directory and exits without
modifying notes.

Requirements: mnemonic, jq, Bash 4+
USAGE
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

if [[ ${1:-} == "-h" || ${1:-} == "--help" ]]; then
  usage
  exit 0
fi

project=${1:-}
max_passes=${2:-5}

[[ -n "$project" ]] || {
  usage >&2
  exit 64
}
[[ "$max_passes" =~ ^[1-9][0-9]*$ ]] || fail "MAX_PASSES must be a positive integer"

require_command mnemonic
require_command jq

workdir=$(mktemp -d "${TMPDIR:-/tmp}/mnemonic-repair.XXXXXX")
trap 'rm -rf "$workdir"' EXIT

kinds=(
  invalid_frontmatter
  missing_required_field
  missing_summary
  duplicate_slug
  duplicate_alias
  unresolved_link
  ambiguous_link
  empty_body
)

kind_args=()
for kind in "${kinds[@]}"; do
  kind_args+=(--kind "$kind")
done

for ((pass = 1; pass <= max_passes; pass++)); do
  report="$workdir/diagnostics-pass-$pass.json"
  printf 'Diagnostic pass %d/%d for project %s\n' "$pass" "$max_passes" "$project"

  mnemonic --json project doctor "$project" "${kind_args[@]}" >"$report"

  if ! jq -e 'type == "object" and (.issues | type == "array")' "$report" >/dev/null; then
    fail "unexpected diagnostic JSON schema"
  fi

  issue_count=$(jq -r '.total_count // (.issues | length)' "$report")
  [[ "$issue_count" =~ ^[0-9]+$ ]] || fail "unexpected total_count in diagnostic JSON"
  printf 'Found %s issue(s).\n' "$issue_count"

  if [[ "$issue_count" == "0" ]]; then
    final_doctor="$workdir/final-doctor.json"
    mnemonic --json project doctor "$project" >"$final_doctor"
    printf 'Note diagnostics are clean. Broad doctor result:\n'
    jq . "$final_doctor"
    exit 0
  fi

  jq . "$report"

  if [[ -z ${MNEMONIC_REPAIR_HOOK:-} ]]; then
    safe_project=${project//[^a-zA-Z0-9._-]/_}
    output="${PWD}/mnemonic-diagnostics-${safe_project}.json"
    cp "$report" "$output"
    printf 'No MNEMONIC_REPAIR_HOOK is configured. No changes were made.\n' >&2
    printf 'Diagnostic report: %s\n' "$output" >&2
    exit 2
  fi

  if [[ ! -x "$MNEMONIC_REPAIR_HOOK" ]]; then
    fail "MNEMONIC_REPAIR_HOOK is not executable: $MNEMONIC_REPAIR_HOOK"
  fi

  "$MNEMONIC_REPAIR_HOOK" "$project" "$report"
done

fail "issues remain after $max_passes passes"
