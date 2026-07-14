# comparison-chain

Chained relational comparisons (a < b < c) do not test a range

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | syntax |
| Default | enabled |
| Fixable | no |
| Tags | comparison, range |

## Details

Pawn evaluates chained comparisons from left to right:

```pawn
0 < value < 10
```

This compares the first result, 0 or 1, with 10. Write the range test as:

```pawn
0 < value && value < 10
```

No fix is offered because complex expressions may need manual changes.
