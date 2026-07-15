# redundant-parentheses

Reports expression parentheses that do not affect parsing

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | yes |
| Tags | expressions, parentheses, style |

## Details

Parentheses are redundant when removing them preserves Pawn precedence, associativity, argument boundaries, statement syntax, and assignment-condition intent. Macro, uncertain, and malformed syntax is ignored.

## Configuration

```toml
[rules]
redundant-parentheses = "warning"
```

## Examples

### Bad

```pawn
UseValues(first, second)
{
    return first + second;
}

main()
{
    new a = 1, b = 2, c = 3;
    new simple = (a);
    new precedence = (a * b) + c;
    new tighter = a + (b * c);
    UseValues((a), b);
    if ((a))
    {
        return ((simple));
    }
    return simple + precedence + tighter;
}
```

### Good

```pawn
UseValues(first, second)
{
    return first + second;
}

WithDefault(value = (1, 2))
{
    return value;
}

main()
{
    new a = 1, b = 2, c = 3;
    new grouped = a * (b + c);
    new rightAssociative = a - (b - c);
    new explicitComparison = (a < b) < c;
    new commaValue = (a, b);
    UseValues((a, b), c);
    if ((a = b))
    {
        return -(a + b);
    }
    (Float:a);
    return grouped + rightAssociative + explicitComparison + commaValue;
}

#define SQUARE(%0) ((%0) * (%0))
```
