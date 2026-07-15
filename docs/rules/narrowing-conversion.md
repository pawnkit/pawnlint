# narrowing-conversion

Reports values that may not fit in packed characters

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | conversions, packed, characters, ranges |

## Details

Assignments through packed-character selection store only values from 0 through 255. The rule reports definite constant and bounded ranges outside that range. Unknown values, ordinary cell subscripts, macros, and uncertain expressions are ignored.

## Configuration

```toml
[rules]
narrowing-conversion = "warning"
```

## Examples

### Bad

```pawn
Check(value, condition)
{
    new packed[4 char];
    packed{0} = 256;
    packed{1} = -1;
    packed{2} = condition ? 0 : 300;
    packed /* storage */ {3} = value & 0x1FF;
}
```

### Good

```pawn
Check(value, condition)
{
    new packed[6 char];
    new ordinary[1];
    packed{0} = 0;
    packed{1} = 255;
    packed{2} = condition != 0;
    packed{3} = value & 0xFF;
    packed{4} = value >>> 24;
    packed{5} = value;
    ordinary[0] = 1000;
}
```
