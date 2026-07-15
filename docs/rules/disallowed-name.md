# disallowed-name

Reports declarations denied by configured name policies

| | |
| --- | --- |
| Category | restriction |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | naming, restriction, policy, identifiers |

## Details

Configured policies deny exact names or regular-expression matches for selected symbol kinds, scopes, storage classes, and tags. Exclusions override a policy. Callbacks and natives require explicit opt-in.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `policies` | object-list | `[]` | — | Name deny-list policy objects |
