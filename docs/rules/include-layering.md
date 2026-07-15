# include-layering

Reports dependencies outside a source layer's allowlist

| | |
| --- | --- |
| Category | restriction |
| Severity | error |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | includes, architecture, project, policy |

## Details

Configured include path globs define the dependencies allowed for matching files. Use path overrides to assign a different allowlist to each source layer.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `allow` | string-list | `[]` | — | Include path glob patterns allowed in the layer |
