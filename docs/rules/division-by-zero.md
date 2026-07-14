# division-by-zero

Reports division or remainder by a constant zero

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | constants, arithmetic, control-flow |

## Details

Division and remainder by zero are invalid. The rule reports only operands that can be evaluated with certainty.
