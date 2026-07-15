# boolean-complexity

Reports boolean expressions with too many logical operators

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | complexity, boolean, maintainability |

## Details

Each maximal expression chain counts its && and || operators. Parentheses, boolean negation, and tag wrappers remain part of the same chain, while nested ternary branches and comparisons are checked independently. Inactive and uncertain syntax is ignored.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `3` | minimum 1; maximum 1000 | Maximum logical operators per expression |
