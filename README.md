# pawnlint

Static analysis and linting for Pawn v3 (SA-MP and open.mp).

`pawnlint` parses with [`pawn-parser`](https://github.com/pawnkit/pawn-parser).
The Pawn compiler is still the source of truth for compilation.

## Features

- Syntax, semantic, control-flow, and include-graph analysis
- 53 built-in rules across correctness, suspicious, performance,
  maintainability, and open.mp categories
- Safe automatic fixes, inline suppressions, and configurable profiles
- Text, JSON, JSONL, SARIF, and GitHub Actions output
- Stdin support for editor and CI integration

Cross-file analysis and fix coverage are still limited — see
[Limitations](docs/limitations.md).

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
| `pawnlint --list-rules` | List every rule. |
| `pawnlint --explain rule-id` | Show a rule's full explanation. |
| `pawnlint --init-config` | Write a default `pawnlint.toml`. |
| `pawnlint --enable id --disable id paths...` | Toggle specific rules. |
| `pawnlint --format text\|compact\|json\|jsonl\|sarif\|github paths...` | Choose output format. |
| `pawnlint --diff paths...` | Preview available fixes. |
| `pawnlint --fix-safe paths...` | Apply only machine-safe fixes. |

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | No finding reached the failure threshold. |
| `1` | A finding reached the failure threshold, or `--diff` found changes. |
| `2` | Invalid arguments or configuration. |
| `3` | Project analysis failed. |

Errors fail by default. Warnings fail too when `warnings-as-errors = true`.

## Documentation

- [Rules](docs/rules/index.md)
- [Configuration](docs/configuration.md)
- [Suppressions](docs/suppression.md)
- [Architecture](docs/architecture.md)
- [Limitations](docs/limitations.md)
- [Contributing](docs/contributing.md)
