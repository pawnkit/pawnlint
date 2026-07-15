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
native sscanf(const input[], const format[], {Float,_}:...);

main()
{
    new playerid;
    new level;
    sscanf("5 10", "dd", playerid);
    return level;
}
```

### Good

```pawn
native sscanf(const input[], const format[], {Float,_}:...);

main()
{
    new playerid;
    new level;
    sscanf("5 10", "dd", playerid, level);
    return playerid + level;
}
```
