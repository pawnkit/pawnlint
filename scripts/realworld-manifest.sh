#!/usr/bin/env bash

realworld_manifest() {
  local workspace corpus_manifest project_manifest
  workspace=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
  corpus_manifest="${PAWN_CORPUS_DIR:-$workspace/../pawn-corpus}/real-world/PROJECTS.tsv"
  project_manifest="$workspace/testdata/realworld/projects.tsv"

  declare -A entries=()
  declare -A configs=()
  while IFS=$'\t' read -r project entry config; do
    entries["$project"]=$entry
    configs["$project"]=$config
  done < <(tail -n +2 "$project_manifest")

  local found=0
  while IFS=$'\t' read -r project repository revision _license _role; do
    if [[ -z ${entries[$project]+x} ]]; then
      continue
    fi
    printf '%s\t%s\t%s\t%s\t%s\n' \
      "$project" "$repository" "$revision" "${entries[$project]}" "${configs[$project]}"
    ((found += 1))
  done < <(tail -n +2 "$corpus_manifest")

  if ((found != ${#entries[@]})); then
    echo "pawnlint real-project selection contains an unknown project" >&2
    return 1
  fi
}
