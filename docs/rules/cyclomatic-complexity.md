# cyclomatic-complexity

Reports functions with too many independent control-flow paths

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | complexity, control-flow, maintainability |

## Details

Complexity starts at one and increases for conditionals, loops, non-default switch cases, ternary expressions, and short-circuit boolean operators. Inactive and uncertain conditional-compilation branches are ignored.

## Configuration

```toml
[rules]
cyclomatic-complexity = "warning"
```

Set options under `[rules.cyclomatic-complexity]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `10` | minimum 1; maximum 10000 | Maximum permitted complexity |

## Examples

### Bad

```pawn
ComplexFlow(value, limit)
{
    if (value > 0 && limit > 0)
    {
        for (new index = 0; index < limit; index++)
        {
            value += index ? 1 : 2;
        }
    }
    switch (value)
    {
        case 1: return 1;
        case 2: return 2;
        default: return 0;
    }
}

BooleanPaths(first, second, third)
{
    return first || second || third ? 1 : 0;
}
```

### Good

```pawn
Simple(value)
{
    if (value)
    {
        return value;
    }
    return 0;
}

SwitchValue(value)
{
    switch (value)
    {
        case 1: return 1;
        case 2: return 2;
        default: return 0;
    }
}
```
