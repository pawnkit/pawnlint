# maximum-nesting

Reports functions with deeply nested control statements

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | complexity, nesting, maintainability |

## Details

Nesting depth increases for if, loop, and switch statements. Else-if chains remain at one level. Inactive and uncertain conditional-compilation branches are ignored.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `4` | minimum 1; maximum 1000 | Maximum permitted nesting depth |
