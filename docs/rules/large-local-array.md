# large-local-array

Reports large automatic arrays allocated on the Pawn stack

| | |
| --- | --- |
| Category | performance |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | arrays, stack, memory, performance |

## Details

Automatic local arrays consume stack cells on every active call. The rule reports arrays whose constant total capacity reaches the configured threshold and skips static or unresolved dimensions.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `threshold` | integer | `1024` | minimum 1 | Minimum local array capacity to report |
