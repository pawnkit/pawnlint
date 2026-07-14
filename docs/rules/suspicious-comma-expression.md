# suspicious-comma-expression

The comma operator chains sub-expressions; it is rarely intended in statements or returns

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | syntax |
| Default | enabled |
| Fixable | no |
| Tags | comma, readability |

## Details

The comma operator evaluates each expression but keeps only the last value:

```pawn
return First(), Second();
```

The rule checks comma expressions used as statements or return values. It does
not report argument lists, declarations, initializers, or for clauses. No fix
is offered because the intended result is unknown.
