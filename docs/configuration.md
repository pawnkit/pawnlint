# Configuration

`pawnlint` searches upward from the current directory for a config file, in
order: `pawnlint.toml`, `pawnlint.yaml`, `pawnlint.yml`, `pawnlint.json`.
Use `--config <path>` to select a file directly (format is chosen by
extension), or `--init-config` to print a default TOML configuration.

## Example

```toml
profile = "recommended"
target = "openmp"

include = ["gamemodes/**/*.pwn", "includes/**/*.inc"]
exclude = ["vendor/**", "generated/**"]

defines = []
include-paths = []
api-metadata = ["pawnlint-api.json"]

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

TOML, JSON, and YAML are equivalent — same keys, same nesting, only the
syntax differs. The example above in JSON:

```json
{
  "profile": "recommended",
  "target": "openmp",
  "include": ["gamemodes/**/*.pwn", "includes/**/*.inc"],
  "exclude": ["vendor/**", "generated/**"],
  "defines": [],
  "include-paths": [],
  "api-metadata": ["pawnlint-api.json"],
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
include: ["gamemodes/**/*.pwn", "includes/**/*.inc"]
exclude: ["vendor/**", "generated/**"]
defines: []
include-paths: []
api-metadata: ["pawnlint-api.json"]
lint:
  warnings-as-errors: false
  max-diagnostics: 0
rules:
  discarded-expression: off
```

Unknown fields are always a configuration error. TOML and YAML report every
unknown field at once; JSON reports the first one found.

## Variants

A name absent from `defines` is *uncertain*, not confidently undefined — so
`#if defined(NAME)` code is skipped rather than analyzed under the wrong
assumption. That means a target- or feature-specific block, e.g.
`#if defined SAMP` / `#if defined OPENMP`, is never analyzed unless that exact
name is in `defines` for the run.

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
  only reported if *every* variant agreed the directive went unused — it may
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
      "parameters": [{"name": "value"}]
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
      "parameters": [{"name": "result", "reference": true, "output": true}]
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
    }
  },
  "constants": {
    "PLUGIN_LIMIT": {}
  }
}
```

Function ownership contracts apply only to unambiguous project definitions.
`release` marks an owned return value. Scalar input parameters may use
`ownership` values `borrowed` or `transferred`. Native entries also support
`deprecated`, `mustUse`, `requiresBefore`, `formatParameter`, and `buffers`.
Invalid fields and relations are configuration errors.

## Profiles

| Profile | Purpose |
| --- | --- |
| `recommended` | Default low-noise rules. |
| `strict` | More suspicious, maintainability, and target/API checks. |
| `all` | Every implemented rule. |

`strict` includes the native/API/migration rules that apply to your `target`
(see [Precedence](#precedence)) — for example, `unimplemented-function` only
reports under `--target openmp`, and `target-native-availability` only
reports under `--target samp`. Setting `--profile strict` without the
matching `--target` simply means those specific rules stay quiet.

## Rule settings

Set a rule to `error`, `warning`, `info`, `hint`, or `off`.
Rule tables accept only the options documented on that rule's page. Option
types, choices, and ranges are validated before linting.

Configuration errors include unknown fields, rule IDs, profiles, targets, and
severity names.

### Naming conventions

```toml
[rules.naming-convention]
severity = "warning"
conventions = [
  { kinds = ["function"], case = "PascalCase", exclude = ["^main$"] },
  { kinds = ["global"], storage = ["const"], case = "UPPER_SNAKE_CASE" },
  { kinds = ["local", "parameter"], case = "camelCase" }
]
```

Conventions are checked in order. Selectors support `kinds`, `scopes`,
`storage`, and `tags`. Policies support `case`, `prefix`, `suffix`, `pattern`,
and `exclude`. Callbacks and natives require `include-callbacks` or
`include-natives`.

```toml
[rules.disallowed-name]
severity = "warning"
policies = [
  { kinds = ["local", "parameter"], names = ["foo", "bar"] },
  { patterns = ["^temp_"], exclude = ["^temporaryAllowed$"] }
]
```

Disallowed-name policies use the same selectors and opt-ins. Each policy needs
`names` or `patterns` and may provide a `reason`.

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
