# duplicate-switch-case

Reports repeated constant values in one switch statement

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | constants, switch, semantic |

## Details

Two case values with the same constant can never select different branches. Case ranges are skipped until range overlap analysis is available.
