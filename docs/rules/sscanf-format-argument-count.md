# sscanf-format-argument-count

Reports sscanf() calls whose format string and argument count differ

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | sscanf, format, arguments |

## Details

sscanf's format string lists one specifier per output argument (aside from
zero-argument directives like `p<delim>` and `{skipped}` groups). A mismatch
between the specifier count and the argument list either fails silently or
crashes at runtime:

```pawn
sscanf(params, "dd", id); // "dd" needs 2 arguments, only 1 given
```

The rule only checks calls with a literal format string and recognizes the
documented specifier set; format strings using unrecognized letters are
skipped rather than guessed at.

## Configuration

```toml
[rules]
sscanf-format-argument-count = "error"
```

## Examples

### Bad

```pawn
main()
{
    new id, level, name[24];

    sscanf("", "dd", id);
    sscanf("", "d", id, level);
    sscanf("", "s[24]d", name);
    sscanf("", "{s[4]}s[24]", name, level);
}
```

### Good

```pawn
main()
{
    new id, level, name[24], reason[128];

    sscanf("", "d", id);
    sscanf("", "dd", id, level);
    sscanf("", "s[24]d", name, level);
    sscanf("", "{s[4]}s[24]", name);
    sscanf("", "ds[128]", id, reason);
    sscanf("", "s[24]C(a)C()", name, level, id);
    sscanf("", "p<,>fff{fff}", id, level, id);
    sscanf("", "a<s[24]>[32]", name);
    sscanf("", "dD(0)", id, level);
    sscanf("", "u", id);

    new fmt[8] = "d";
    sscanf("", fmt, id);
}
```
