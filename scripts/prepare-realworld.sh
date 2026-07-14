#!/usr/bin/env bash
set -euo pipefail

workspace=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
destination=${1:-/tmp/pawnlint-corpus}
manifest="$workspace/testdata/realworld/corpora.tsv"
sampctl=${SAMPCTL:-sampctl}

mkdir -p "$destination"
while IFS=$'\t' read -r name repository commit entry config; do
  directory="$destination/$name"
  if [[ ! -d "$directory/.git" ]]; then
    git clone "$repository" "$directory"
  fi
  git -C "$directory" fetch origin "$commit"
  git -C "$directory" checkout --detach "$commit"
  if [[ -f "$directory/pawn.json" ]]; then
    if ! command -v "$sampctl" >/dev/null 2>&1; then
      echo "sampctl is required to prepare $name" >&2
      exit 1
    fi
    (cd "$directory" && "$sampctl" ensure)
  fi
done < "$manifest"
