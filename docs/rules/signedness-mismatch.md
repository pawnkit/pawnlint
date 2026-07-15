# signedness-mismatch

Reports packed-character comparisons with negative values

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | signedness, packed, characters, comparisons |

## Details

Packed-character selection produces values from 0 through 255. Comparing one with a definitely negative cell value usually indicates a sentinel or storage mistake. Unknown ranges, ordinary cell subscripts, macros, and uncertain expressions are ignored.
