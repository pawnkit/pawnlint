# signedness-mismatch

Reports packed-character comparisons with negative values

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | signedness, packed, characters, comparisons |

## Details

Packed-character selection produces values from 0 through 255. Comparing one with a definitely negative cell value usually indicates a sentinel or storage mistake. Unknown ranges, ordinary cell subscripts, macros, and uncertain expressions are ignored.

## Configuration

```toml
[rules]
signedness-mismatch = "warning"
```

## Examples

### Bad

```pawn
Check()
{
    new packed[3 char];
    if (packed{0} == -1) return 1;
    if (-2 < packed{1}) return 2;
    if (packed{2} >= -10) return 3;
    return 0;
}
```

### Good

```pawn
Check(value)
{
    new packed[3 char];
    new ordinary[1];
    if (packed{0} == 0) return 1;
    if (packed{1} <= 255) return 2;
    if (packed{2} == value) return 3;
    if (ordinary[0] == -1) return 4;
    if ((value & 0xFF) == packed{0}) return 5;
    return 0;
}
```
