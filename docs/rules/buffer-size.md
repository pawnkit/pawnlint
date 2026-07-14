# buffer-size

Reports native size arguments larger than a declared buffer

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | buffer, arrays, native, api |

## Details

Official native declarations link output arrays to capacity parameters with defaults such as sizeof(buffer). The rule reports only direct array arguments with one known dimension and a definite oversized value.
