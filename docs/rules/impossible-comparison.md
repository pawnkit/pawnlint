# impossible-comparison

Reports comparisons that cannot produce both results

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | comparisons, ranges, conditions, semantic |

## Details

Definite ranges from boolean expressions, remainders, bit masks, unsigned shifts, and conditional expressions prove when a comparison is always true or false. Unknown, floating-point, overflowing, macro-derived, and malformed expressions are ignored.
