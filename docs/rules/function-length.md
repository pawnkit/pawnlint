# function-length

Reports functions spanning too many source lines

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | size, functions, maintainability |

## Details

Physical lines are counted from the function signature through the end of its body, including blank and comment lines. Inactive and uncertain conditional-compilation branches are excluded.

## Configuration

```toml
[rules]
function-length = "warning"
```

Set options under `[rules.function-length]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `100` | minimum 1; maximum 1000000 | Maximum physical lines per function |

## Examples

### Bad

```pawn
LongFunction(value)
{
    // Comments count as physical lines.

    if (value)
    {
        value--;
    }

    return value;
}

LongSignature(
    first,
    second,
    third,
    fourth,
    fifth)
{
    return first + second + third + fourth + fifth;
}
```

### Good

```pawn
Boundary(value)
{
    if (value)
    {
        return value;
    }
    return 0;
}

OneLine() { return 1; }
```
