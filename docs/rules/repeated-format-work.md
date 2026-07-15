# repeated-format-work

Reports invariant formatting repeated before a buffer is used

| | |
| --- | --- |
| Category | performance |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | strings, formatting, loops, performance |

## Details

Formatting the same values into an untouched local buffer on every loop iteration repeats work when the buffer is only consumed after the loop. The rule checks direct format, Format, and strformat statements with invariant inputs and ignores mutable substitutions, conditional calls, macros, uncertain loops, shadowed natives, and buffers accessed in the loop.

## Configuration

```toml
[rules]
repeated-format-work = "warning"
```

## Examples

### Bad

```pawn
BuildLabel(count, value)
{
    new label[32];
    for (new i; i < count; i++)
    {
        format(label, sizeof label, "Value: %d", value);
    }
    return label[0];
}
```

### Good

```pawn
BuildLabel(count, value)
{
    new label[32];
    format(label, sizeof label, "Value: %d", value);
    for (new i; i < count; i++)
    {
        printf("%s", label);
    }
    return label[0];
}
```
