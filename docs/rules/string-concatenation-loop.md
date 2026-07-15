# string-concatenation-loop

Reports strcat calls that repeatedly scan a growing buffer

| | |
| --- | --- |
| Category | performance |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | strings, loops, strcat, performance |

## Details

Appending to the same string with strcat on every loop iteration repeatedly scans the growing destination. The rule checks unconditional calls on one-dimensional local buffers that survive the loop and ignores reset, accessed, conditional, macro-derived, uncertain, self-appending, and shadowed cases.
