# Configuration

`pawnlint` searches upward from the current directory for `pawnlint.toml`,
`pawnlint.yaml`, `pawnlint.yml`, or `pawnlint.json`, in that order.
`--config <path>` selects a file directly. `--init-config` writes a default
TOML file and will not overwrite an existing file.

## Example

```toml
profile = "recommended"
target = "openmp"
presets = ["config/policy.toml"]

include = ["gamemodes/**/*.pwn", "includes/**/*.inc"]
exclude = ["vendor/**", "generated/**"]

defines = []
include-paths = []
api-metadata = ["pawnlint-api.json"]
baseline = "pawnlint-baseline.json"
cache = ".pawnlint-cache"

[lint]
warnings-as-errors = false
max-diagnostics = 0

[rules]
discarded-expression = "off"

# [rules.some-configurable-rule]
# severity = "warning"
# threshold = 20
```

- `max-diagnostics = 0` means unlimited.
- `defines` names are treated as present by `defined(NAME)` conditions.
- Relative `include-paths` entries resolve from the configuration file.
- Relative `api-metadata` entries resolve the same way; later files override
  earlier entries for the same key.
- A relative `baseline` path resolves from the configuration file.

## Cache

Set `cache` to reuse diagnostics when resolved sources, effective configuration,
API metadata, pawn-parser, and pawnlint rule code are unchanged. Relative paths
resolve from the configuration file. Cache errors are treated as misses, and the
project graph is still rebuilt to validate includes. The directory can be deleted
at any time.

## Presets

`presets` loads shared policy files relative to the declaring file. Presets may
set `profile`, `lint`, `rules`, and `overrides`. Later presets override earlier
ones, then the local config wins. Only listed files are loaded; source, build,
target, and API context remain in the local config.

## Rule aliases

Renamed rule IDs continue to work in configs, CLI overrides, and suppressions.
pawnlint reports the replacement ID; new configuration should use it directly.

## Baselines

Set `baseline` to suppress matching existing findings. Run `pawnlint
--generate-baseline paths...` to replace it or `pawnlint --prune-baseline
paths...` to remove resolved entries. `--baseline <path>` overrides the
configured path. Parser and internal errors are never baselined.

## Timings

Run `pawnlint --timings paths...` to print parsing, semantic, control-flow,
project, rule, output, and total durations to stderr. Project durations include
parsing and semantics; rule durations are cumulative across concurrent files.

## Builds

`builds` lets a build system provide compiler contexts, then invoke `pawnlint`
without source arguments. pawnlint does not interact with package managers.

```toml
[[builds]]
name = "main"
entry = "gamemodes/main.pwn"
working-directory = "."
files = ["gamemodes/**", "includes/**"]
exclude = ["includes/generated/**"]
include-paths = ["dependencies/pawn-stdlib", "dependencies/project-api"]
defines = ["OPENMP", "FEATURE_X"]
target = "openmp"
```

- `name` and `entry` are required; names must be unique.
- Paths are relative to `working-directory`, which defaults to the config directory.
- Top-level include paths and defines are inherited by each build.
- `defines` is the complete initial set; absent names are undefined.
- `files` selects reachable includes to lint; unselected includes still provide context.
- The entry is always linted; `exclude` only applies to additional files.
- Build `target` overrides the config target; `--target` overrides all builds.
- Shared diagnostics are deduplicated across builds.
- `builds` and `variants` cannot be combined.
- TOML, JSON, and YAML use the same field names.

## Formats

TOML, JSON, and YAML use the same keys and nesting. The example above in JSON:

```json
{
  "profile": "recommended",
  "target": "openmp",
  "presets": ["config/policy.toml"],
  "include": ["gamemodes/**/*.pwn", "includes/**/*.inc"],
  "exclude": ["vendor/**", "generated/**"],
  "defines": [],
  "include-paths": [],
  "api-metadata": ["pawnlint-api.json"],
  "baseline": "pawnlint-baseline.json",
  "cache": ".pawnlint-cache",
  "lint": {
    "warnings-as-errors": false,
    "max-diagnostics": 0
  },
  "rules": {
    "discarded-expression": "off"
  }
}
```

and in YAML:

```yaml
profile: recommended
target: openmp
presets: ["config/policy.toml"]
include: ["gamemodes/**/*.pwn", "includes/**/*.inc"]
exclude: ["vendor/**", "generated/**"]
defines: []
include-paths: []
api-metadata: ["pawnlint-api.json"]
baseline: pawnlint-baseline.json
cache: .pawnlint-cache
lint:
  warnings-as-errors: false
  max-diagnostics: 0
rules:
  discarded-expression: off
```

Unknown fields are always a configuration error. TOML and YAML report every
unknown field at once; JSON reports the first one found.

## Variants

A name absent from `defines` is uncertain. Pawnlint skips its conditional code
instead of treating it as undefined. Add names such as `SAMP` or `OPENMP` to
analyze their blocks.

`variants` re-runs the full build and lint pass once per entry, each with its
own `defines`, then merges the results:

```toml
[[variants]]
name = "openmp"
defines = ["OPENMP"]

[[variants]]
name = "samp"
defines = ["SAMP"]
```

- Diagnostics are deduplicated by (rule, file, range), so code shared by every
  variant is reported once.
- Exception: `unknown-suppression`'s "unused suppression directive" hint is
  only reported if every variant agreed the directive went unused; it may
  guard code active under just one variant.
- Each variant name must be non-empty and unique.
- `variants` cannot be combined with `builds`.
- With no `variants` configured, behavior is unchanged: a single pass using
  the top-level `defines`.

## Overrides

`overrides` applies a `[rules]`-shaped table only to files whose
project-relative path matches at least one glob in `paths` (same syntax as
`include`/`exclude`):

```toml
[[overrides]]
paths = ["testdata/**", "generated/**"]
[overrides.rules]
unused-local = "off"
large-local-array = "hint"
```

- Each override needs at least one path and at least one rule.
- Later overrides win over earlier ones.
- Overrides take priority over the top-level `[rules]` table for the same
  rule ID on a matching path.
- A rule an override doesn't mention keeps its base severity for that path.

## API metadata

Use JSON to describe plugin or project APIs:

```json
{
  "callbacks": {
    "OnPluginEvent": {
      "returnTag": "bool",
      "parameters": [{"name": "value", "taintSource": "player-input"}]
    }
  },
  "natives": {
    "Plugin_Init": {},
    "Plugin_Open": {
      "returnTag": "PluginHandle",
      "release": "Plugin_Close",
      "requiresBefore": ["Plugin_Init"]
    },
    "Plugin_Close": {
      "parameters": [{"name": "handle", "tag": "PluginHandle"}]
    },
    "Plugin_Parse": {
      "parameters": [{"name": "result", "reference": true, "output": true, "taintSource": "network-input"}]
    },
    "Plugin_Query": {
      "parameters": [{"name": "query", "arrayRank": 1, "const": true, "taintSink": "sql"}]
    },
    "Plugin_Clamp": {
      "pure": true,
      "parameters": [{"name": "value"}]
    }
  },
  "functions": {
    "OpenLog": {
      "returnTag": "File",
      "release": "CloseLog"
    },
    "CloseLog": {
      "parameters": [{"name": "file", "tag": "File", "ownership": "transferred"}]
    },
    "InspectLog": {
      "parameters": [{"name": "file", "tag": "File", "ownership": "borrowed"}]
    },
    "Normalize": {
      "pure": true,
      "parameters": [{"name": "value"}]
    }
  },
  "constants": {
    "PLUGIN_LIMIT": {}
  }
}
```

Function contracts apply only to unambiguous project definitions. `release`
marks an owned return value, `ownership` accepts `borrowed` or `transferred`,
and `pure` marks deterministic calls without observable effects. Parameter
`taintSource` and `taintSink` labels use lowercase names such as `player-input`,
`sql`, `command`, `file`, and `format`. Native entries also support `deprecated`,
`mustUse`, `requiresBefore`, `formatParameter`, and `buffers`. Invalid fields and
relations are configuration errors.

## Profiles

| Profile | Purpose |
| --- | --- |
| `recommended` | Default low-noise rules. |
| `strict` | More suspicious, maintainability, and target/API checks. |
| `all` | Every implemented rule. |

`strict` includes the native/API/migration rules that apply to your `target`
(see [Precedence](#precedence)). For example, `unimplemented-function` only
reports under `--target openmp`, and `target-native-availability` only
reports under `--target samp`. Setting `--profile strict` without the
matching `--target` simply means those specific rules stay quiet.

## Rule settings

Set a rule to `error`, `warning`, `info`, `hint`, or `off` under `[rules]`.
Each page in the [rule index](rules/index.md) lists its options, defaults, and
examples. Rule tables accept only those documented options.

Configuration errors include unknown fields, rule IDs, profiles, targets, and
severity names.

## External rules

Configure versioned process rules with `[[external-rules]]`. See [External rules](external-rules.md) for the protocol and limits.

## Precedence

Highest priority first:

1. `--disable`
2. `--enable`
3. `--profile` and `--target`
4. `[rules]`
5. Profile defaults

## Globs

- `**` matches any number of path segments.
- `*` matches within one segment.
- `?` matches one character.

Paths use `/` separators and are matched relative to the project directory.

## Stdin

```sh
pawnlint --stdin --stdin-filename gamemodes/main.pwn
```

Configuration discovery still starts from the current directory.
