#!/usr/bin/env bash
set -euo pipefail

workspace=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
corpora=${1:-/tmp/pawnlint-corpus}
source "$workspace/scripts/realworld-manifest.sh"
manifest=$(realworld_manifest)

while IFS=$'\t' read -r name _repository _commit _entry _config; do
  directory="$corpora/$name"
  if [[ ! -f "$directory/pawn.json" && ! -f "$directory/pawn.yaml" ]]; then
    continue
  fi
  if ! sampctl ensure --dir "$directory"; then
    if [[ ! -d "$directory/dependencies" ]]; then
      echo "$name: dependency installation failed before producing includes" >&2
      exit 1
    fi
    echo "$name: continuing with includes installed before the runtime resource failure" >&2
  fi
done <<< "$manifest"
