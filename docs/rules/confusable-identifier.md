# confusable-identifier

Reports visible declarations with visually confusable names

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | naming, suspicious, identifiers |

## Details

Pawn identifiers are ASCII. The rule reports declarations whose names differ but become identical after normalizing the visually ambiguous groups 0/O/o and 1/I/l. Only definite declarations visible in the same lexical context are compared.
