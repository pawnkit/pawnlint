# float-equality

Reports Float values compared with == or !=

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | float, comparison, semantic |

## Details

Float values accumulate rounding error, so `==` and `!=` rarely mean
what they appear to:

```pawn
if (Float:distance == 0.0)
```

Use a tolerance comparison, or round both sides with `floatround` first if an
exact match is really intended:

```pawn
if (floatround(distance) == floatround(target))
```

The rule does not report when either side is a direct `floatround(...)` call,
since rounding first is the standard way to compare floats exactly. No fix is
offered because the correct tolerance or rounding style depends on context.

## Configuration

```toml
[rules]
float-equality = "warning"
```

## Examples

### Bad

```pawn
Float:GetSpeed()
{
    return 1.0;
}

main()
{
    new Float:distance;
    new Float:target;
    if (distance == 0.0)
    {
    }
    if (distance != target)
    {
    }
    if (Float:1 == Float:2)
    {
    }
    if (GetSpeed() == 0.0)
    {
    }
}
```

### Good

```pawn
main()
{
    new value;
    new Float:distance;
    new Float:target;
    if (value == 5)
    {
    }
    if (value != 0)
    {
    }
    if (distance < target)
    {
    }
    if (distance >= target)
    {
    }
    if (floatround(distance) == floatround(target))
    {
    }
}
```
