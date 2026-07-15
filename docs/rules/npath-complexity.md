# npath-complexity

Reports functions with too many acyclic execution paths

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | complexity, control-flow, maintainability |

## Details

Alternative paths add and sequential branching statements multiply. Loops add an exit path, while short-circuit operators and ternaries add expression alternatives. Counts saturate safely and ignore inactive or uncertain conditional-compilation branches.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `200` | minimum 1; maximum 999999999 | Maximum permitted NPath complexity |
