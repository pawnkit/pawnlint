# string-concatenation-loop

Reports strcat calls that repeatedly scan a growing buffer

| | |
| --- | --- |
| Category | performance |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | strings, loops, strcat, performance |

## Details

Appending to the same string with strcat on every loop iteration repeatedly scans the growing destination. The rule checks unconditional calls on one-dimensional local buffers that survive the loop and ignores reset, accessed, conditional, macro-derived, uncertain, self-appending, and shadowed cases.

## Configuration

```toml
[rules]
string-concatenation-loop = "warning"
```

## Examples

### Bad

```pawn
BuildString(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        strcat(output, piece, sizeof output);
    }
    Consume(output);
}

BuildDefaultLimit(limit, const piece[])
{
    new output[256] = "";
    do {
        strcat(output, piece);
    } while (--limit > 0);
    Consume(output);
}

BuildDelimited(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        strcat(output, piece, sizeof output);
        strcat(output, ",", sizeof output);
    }
    Consume(output);
}
```

### Good

```pawn
BuildScratch(limit, const piece[])
{
    for (new i; i < limit; i++) {
        new output[256] = "";
        strcat(output, piece, sizeof output);
        Consume(output);
    }
}

BuildReset(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        output[0] = EOS;
        strcat(output, piece, sizeof output);
    }
    Consume(output);
}

BuildConsumed(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        strcat(output, piece, sizeof output);
        Consume(output);
    }
}
// …
```
