# function-length

Reports functions spanning too many source lines

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | size, functions, maintainability |

## Details

Physical lines are counted from the function signature through the end of its body, including blank and comment lines. Inactive and uncertain conditional-compilation branches are excluded.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `100` | minimum 1; maximum 1000000 | Maximum physical lines per function |
