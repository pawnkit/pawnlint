# maximum-nesting

Reports functions with deeply nested control statements

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | complexity, nesting, maintainability |

## Details

Nesting depth increases for if, loop, and switch statements. Else-if chains remain at one level. Inactive and uncertain conditional-compilation branches are ignored.

## Configuration

```toml
[rules]
maximum-nesting = "warning"
```

Set options under `[rules.maximum-nesting]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `4` | minimum 1; maximum 1000 | Maximum permitted nesting depth |

## Examples

### Bad

```pawn
DeepLoop(value)
{
    while (value)
    {
        if (value > 1)
        {
            for (new index = 0; index < value; index++)
            {
                value -= index;
            }
        }
    }
    return value;
}
// …
```

### Good

```pawn
Allowed(value)
{
    while (value)
    {
        if (value > 1)
        {
            value--;
        }
    }
    return value;
}

ElseIf(value)
{
    if (value == 1)
    {
        return 1;
    }
    else if (value == 2)
    {
        return 2;
    }
    else if (value == 3)
    {
        return 3;
    }
    return 0;
}
```
