# naming-convention

Reports declarations that violate configured naming policies

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | naming, style, policy, identifiers |

## Details

Ordered conventions select symbols by kind, scope, storage, and tag. The first matching convention checks case, prefix, suffix, and an optional regular expression. Exclusion expressions suppress matching names. Callbacks and natives require explicit opt-in.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `conventions` | object-list | `[]` | — | Ordered naming selector and policy objects |
