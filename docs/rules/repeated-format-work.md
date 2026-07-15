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
CheckFormat(limit, value)
{
    new output[64];
    new total;
    for (new i; i < limit; i++) {
        format(output, sizeof output, "value %d", value);
        total += i;
    }
    Consume(output);
    return total;
}

CheckStrformat(limit)
{
    new output[64];
    while (limit-- > 0) {
        strformat(output, sizeof output, false, "ready");
    }
    Consume(output);
}

CheckOpenMPFormat(limit)
{
    new output[64];
    do {
        Format(output, sizeof output, "ready");
    } while (--limit > 0);
    Consume(output);
}
```

### Good

```pawn
CheckChanging(limit)
{
    new output[64];
    for (new i; i < limit; i++) {
        format(output, sizeof output, "value %d", i);
    }
    Consume(output);
}

CheckConsumed(limit, value)
{
    new output[64];
    for (new i; i < limit; i++) {
        format(output, sizeof output, "value %d", value);
        Consume(output);
    }
}

CheckConditional(limit, value)
{
    new output[64];
    for (new i; i < limit; i++) {
        if (i > 2) {
            format(output, sizeof output, "value %d", value);
        }
    }
    Consume(output);
}
// …
```
