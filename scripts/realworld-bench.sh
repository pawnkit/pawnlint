#!/usr/bin/env bash
set -euo pipefail

workspace=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
corpora=${1:-/tmp/pawnlint-corpus}
output=${2:-/tmp/pawnlint-realworld-results}
manifest="$workspace/testdata/realworld/corpora.tsv"
binary="$output/pawnlint"
stats="$output/realworldstats"

for command in git jq /usr/bin/time; do
  command -v "$command" >/dev/null
done

mkdir -p "$output"
(cd "$workspace" && go build -o "$binary" ./cmd/pawnlint && go build -o "$stats" ./cmd/realworldstats)

while IFS=$'\t' read -r name repository commit entry config; do
  directory="$corpora/$name"
  actual=$(git -C "$directory" rev-parse HEAD)
  if [[ "$actual" != "$commit" ]]; then
    echo "$name: expected $commit, found $actual" >&2
    exit 1
  fi
  benchmark_config="$directory/.pawnlint-benchmark.toml"
  cp "$workspace/testdata/realworld/$config" "$benchmark_config"
  diagnostics="$output/$name.diagnostics.json"
  metrics="$output/$name.metrics.json"
  status_file="$output/$name.status"
  project="$output/$name.project.json"
  report="$output/$name.json"
  /usr/bin/time -f '{"elapsedSeconds":%e,"userSeconds":%U,"systemSeconds":%S,"peakRssKb":%M}' -o "$metrics" \
    bash -c 'status_file=$1; shift; "$@"; status=$?; printf "%s\n" "$status" > "$status_file"' bash "$status_file" \
    "$binary" --config="$benchmark_config" --format=json "$directory/$entry" > "$diagnostics"
  status=$(<"$status_file")
  rm -f "$status_file"
  if (( status > 1 )); then
    exit "$status"
  fi
  "$stats" --root="$directory" --entry="$entry" --config="$benchmark_config" > "$project"
  jq -n \
    --arg name "$name" \
    --arg repository "$repository" \
    --arg commit "$commit" \
    --slurpfile metrics "$metrics" \
    --slurpfile project "$project" \
    --slurpfile diagnostics "$diagnostics" \
    '{name:$name,repository:$repository,commit:$commit,metrics:$metrics[0],project:$project[0],diagnostics:{count:($diagnostics[0]|length),byRule:($diagnostics[0]|group_by(.ruleId)|map({ruleId:.[0].ruleId,count:length}))}}' > "$report"
  jq -c . "$report"
done < "$manifest"
