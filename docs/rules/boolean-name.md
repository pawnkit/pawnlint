# boolean-name

Reports boolean declarations without an allowed prefix

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | naming, style, policy, boolean |

## Details

Ordered policies select declarations with a definite bool tag. The first matching policy requires one configured prefix at a naming boundary. Exclusions override a policy. Callbacks and natives require explicit opt-in.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `policies` | object-list | `[]` | — | Ordered boolean-name selector and prefix objects |
