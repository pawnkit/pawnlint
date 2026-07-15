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

## Configuration

```toml
[rules]
impossible-comparison = "warning"
```

## Examples

### Bad

```pawn
Check(value)
{
    if ((value & 0xFF) > 255)
        return 1;
    if ((value & 3) <= 3)
        return 1;
    if (value % 10 >= 10)
        return 1;
    if (value % 10 < -9)
        return 1;
    if ((value == 0) == 2)
        return 1;
    if ((value == 0) != -1)
        return 1;
    if ((value >>> 8) < 0)
        return 1;
    if ((value ? (value & 3) : (value & 7)) > 7)
        return 1;
    return 0;
}
```

### Good

```pawn
Check(value, Float:ratio)
{
    if ((value & 255) == 255)
        return 1;
    if (value % 10 == 9)
        return 1;
    if ((value == 0) == 1)
        return 1;
    if (value > 2147483647)
        return 1;
    if (ratio > 2.0)
        return 1;
    if (1 < 2)
        return 1;
    if (0 <= value <= 10)
        return 1;
    if (5000 > value > 20000)
        return 1;
    return 0;
}

#define CHECK_RANGE(%0) (((%0) & 3) > 7)
```
