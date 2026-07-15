# swapped-arguments

Reports native arguments whose tags match each other's parameters

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | calls, arguments, api, tags |

## Details

Two positional arguments are reported only when both have one definite tag, both expected parameter tags are distinct, and exchanging the arguments resolves both mismatches. Named, macro-structured, incomplete, and locally overridden calls are skipped.
