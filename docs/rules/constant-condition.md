# constant-condition

Reports if and ternary conditions with a constant result

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | constants, conditions, control-flow |

## Details

A constant condition always selects the same branch. Loops are skipped because constant loop conditions are often intentional.
