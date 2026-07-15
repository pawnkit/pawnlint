# unsafe-string-termination

Reports raw copies used as strings without EOS termination

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | strings, termination, buffers, memcpy |

## Details

A raw memcpy does not append EOS. The rule reports array destinations later passed to a known string parameter in the same function without an intervening EOS assignment or terminating string write. Unresolved, macro-derived, uncertain, malformed, and non-string uses are ignored.

## Configuration

```toml
[rules]
unsafe-string-termination = "warning"
```

## Examples

### Bad

```pawn
Check(const source[])
{
    new first[16];
    new second[16];
    new conditional[16];
    new concatenated[16];
    memcpy(first, source, 0, 16 * 4);
    strlen(first);
    memcpy(second, source, 0, 16 * 4);
    strcmp(second, source);
    memcpy(conditional, source, 0, 16 * 4);
    if (strlen(source) > 0) {
        conditional[15] = EOS;
    }
    strlen(conditional);
    memcpy(concatenated, source, 0, 16 * 4);
    strcat(concatenated, "suffix");
}
```

### Good

```pawn
Check(const source[])
{
    new terminated[16];
    new overwritten[16];
    new binary[16];
    memcpy(terminated, source, 0, 16 * 4);
    terminated[15] = EOS;
    strlen(terminated);
    memcpy(overwritten, source, 0, 16 * 4);
    strmid(overwritten, source, 0, 15, sizeof(overwritten));
    strlen(overwritten);
    memcpy(binary, source, 0, 16 * 4);
}
```
