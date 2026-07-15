# redundant-else

Reports else branches after unconditional control transfer

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | yes |
| Tags | control-flow, branches, style |

## Details

An else is redundant when the preceding branch always returns, breaks, continues, or jumps away. Removing the else keeps the alternative's scope and comments intact. Uncertain and malformed branches are ignored.

## Configuration

```toml
[rules]
redundant-else = "warning"
```

## Examples

### Bad

```pawn
ReturnValue(value)
{
    if (value < 0)
    {
        return 0;
    }
    else
    {
        return value;
    }
}
// …
```

### Good

```pawn
CheckValue(value)
{
    if (value < 0)
    {
        value = 0;
    }
    else
    {
        value++;
    }

    if (value == 1)
    {
        if (value < 0)
            return 0;
    }
    else
    {
        value--;
    }
    return value;
}

#define RETURN_VALUE(%0) if (%0) return 1; else return 0
```
