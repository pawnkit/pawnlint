# argument-value-range

Reports constant arguments outside API parameter bounds

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | calls, arguments, api, contracts, range |

## Details

API metadata can define inclusive minimum and maximum values for scalar input parameters. The rule reports only definite constant violations and skips named, macro-structured, unresolved, and locally overridden calls.
