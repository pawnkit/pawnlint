# narrowing-conversion

Reports values that may not fit in packed characters

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | conversions, packed, characters, ranges |

## Details

Assignments through packed-character selection store only values from 0 through 255. The rule reports definite constant and bounded ranges outside that range. Unknown values, ordinary cell subscripts, macros, and uncertain expressions are ignored.
