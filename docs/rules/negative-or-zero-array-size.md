# negative-or-zero-array-size

Reports array dimensions that evaluate to zero or less

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | constants, arrays, semantic |

## Details

A declared array dimension must be greater than zero. The rule reports only dimensions that can be evaluated with certainty.
