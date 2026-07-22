# pawnlint

Static analysis and linting for Pawn v3 (SA-MP and open.mp).

`pawnlint` parses with [`pawn-parser`](https://github.com/pawnkit/pawn-parser).
The Pawn compiler is still the source of truth for compilation.

## Features

- Syntax, semantic, control-flow, and include-graph analysis
- 115 built-in rules across analysis, performance, API, and policy categories
- Automatic fixes, inline suppressions, baselines, and incremental caching
- Text, JSON, JSONL, SARIF, and GitHub Actions output
- Stdin support for tool and CI integration

## Install

```sh
go install github.com/pawnkit/pawnlint/cmd/pawnlint@latest
```

## Usage

```sh
pawnlint gamemodes/main.pwn
pawnlint gamemodes/
pawnlint --format json gamemodes/
pawnlint --diff gamemodes/
pawnlint --fix gamemodes/
cat main.pwn | pawnlint --stdin --stdin-filename main.pwn
```

The default profile is `recommended`. Use `--profile strict` for more checks.

| Command | Purpose |
| --- | --- |
| `pawnlint --list-rules` | List every rule |
| `pawnlint --explain rule-id` | Explain a rule |
| `pawnlint --init-config` | Write a default `pawnlint.toml` |
| `pawnlint --check-config` | Validate config paths, entries, and includes |
| `pawnlint --enable id --disable id paths...` | Toggle specific rules |
| `pawnlint --format text\|compact\|json\|jsonl\|sarif\|github paths...` | Choose an output format |
| `pawnlint --diff paths...` | Preview available fixes |
| `pawnlint --fix-safe paths...` | Apply machine-safe fixes |
| `pawnlint --generate-baseline paths...` | Replace the configured baseline |
| `pawnlint --prune-baseline paths...` | Remove resolved baseline entries |
| `pawnlint --timings paths...` | Print stage and rule timings |

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | No finding reached the failure threshold. |
| `1` | A finding reached the failure threshold, or `--diff` found changes. |
| `2` | Invalid arguments or configuration. |
| `3` | Project analysis failed. |

Errors fail by default. Warnings fail too when `warnings-as-errors = true`.

## Documentation

- [Rules and configuration](docs/rules/index.md): options and examples for every rule
- [Configuration](docs/configuration.md): profiles, presets, builds, and variants
- [Suppressions](docs/suppression.md)
- [External rules](docs/external-rules.md)
- [Analyzer API](docs/analyzer-api.md)

## Contributing

New rules, false-positive fixes, and small Pawn examples are welcome. See
[CONTRIBUTING.md](CONTRIBUTING.md) for tests and ownership boundaries.
