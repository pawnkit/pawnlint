# npath-complexity

Reports functions with too many acyclic execution paths

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | complexity, control-flow, maintainability |

## Details

Alternative paths add and sequential branching statements multiply. Loops add an exit path, while short-circuit operators and ternaries add expression alternatives. Counts saturate safely and ignore inactive or uncertain conditional-compilation branches.

## Configuration

```toml
[rules]
npath-complexity = "warning"
```

Set options under `[rules.npath-complexity]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `200` | minimum 1; maximum 999999999 | Maximum permitted NPath complexity |

## Examples

### Bad

```pawn
ManyPaths(first, second, third)
{
    if (first)
    {
        first++;
    }
    if (second)
    {
        second++;
    }
    if (third)
    {
        third++;
    }
    while (first)
    {
        if (second)
        {
            break;
        }
        first--;
    }
    return first + second + third;
}
// …
```

### Good

```pawn
Boundary(first, second, third)
{
    if (first)
    {
        first++;
    }
    if (second)
    {
        second++;
    }
    while (third)
    {
        third--;
    }
    return first + second + third;
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
