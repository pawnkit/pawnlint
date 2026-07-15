# suspicious-negation

'!' binds tighter than &/|/^/==/!=, so !x & y is (!x) & y

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | syntax |
| Default | enabled |
| Fixable | no |
| Tags | precedence, negation |

## Details

'!' binds more tightly than bitwise and equality operators:

```pawn
!flags & FLAG_ADMIN
!value == expected
```

The likely forms are `!(flags & FLAG_ADMIN)` and `value != expected`. No fix is
offered because the intended grouping cannot be known.

## Configuration

```toml
[rules]
suspicious-negation = "warning"
```

## Examples

### Bad

```pawn
main()
{
    if (!flags & FLAG_ADMIN)
    {
    }
    if (!value == expected)
    {
    }
    if (!a | b)
    {
    }
    if (!mask ^ tag)
    {
    }
}
```

### Good

```pawn
main()
{
    if (!connected)
    {
    }
    if (flags & FLAG_ADMIN)
    {
    }
    if (value != expected)
    {
    }
    if (!flags)
    {
    }
    if (!(flags & FLAG))
    {
    }
}
```
