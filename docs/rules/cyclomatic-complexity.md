# cyclomatic-complexity

Reports functions with too many independent control-flow paths

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | complexity, control-flow, maintainability |

## Details

Complexity starts at one and increases for conditionals, loops, non-default switch cases, ternary expressions, and short-circuit boolean operators. Inactive and uncertain conditional-compilation branches are ignored.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `10` | minimum 1; maximum 10000 | Maximum permitted complexity |
