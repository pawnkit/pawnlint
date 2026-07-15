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

## Configuration

```toml
[rules]
suspicious-comma-expression = "warning"
```

## Examples

### Bad

```pawn
main()
{
    return Function1(), Function2(), 3;
    a = b, c = d;
    x++, y++;
}
```

### Good

```pawn
main()
{
    return 0;
    a = 1, b = 2;
    foo(a, b, c);
    new list[] = {1, 2, 3};
    for (new i = 0, j = 10; i < j; i++, j--)
    {
    }
}
```
