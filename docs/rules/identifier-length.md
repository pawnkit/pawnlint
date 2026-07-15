# identifier-length

Reports declarations outside configured name-length limits

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | naming, style, policy, identifiers |

## Details

Ordered limits select declarations by kind, scope, storage, and tag. The first matching limit checks minimum and maximum ASCII identifier lengths. Callbacks and natives require explicit opt-in. One-character loop indices can be allowed when their role is definite.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `limits` | object-list | `[]` | — | Ordered identifier-length selector and limit objects |
