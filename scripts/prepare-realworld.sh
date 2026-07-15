#!/usr/bin/env bash
set -euo pipefail

workspace=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
destination=${1:-/tmp/pawnlint-corpus}
manifest="$workspace/testdata/realworld/corpora.tsv"

mkdir -p "$destination"
while IFS=$'\t' read -r name repository commit entry config; do
  directory="$destination/$name"
  if [[ ! -d "$directory/.git" ]]; then
    git clone "$repository" "$directory"
  fi
  git -C "$directory" fetch origin "$commit"
  git -C "$directory" checkout --detach "$commit"
done < "$manifest"
