# duplicate-switch-case

Reports repeated constant values in one switch statement

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | constants, switch, semantic |

## Details

Two case values with the same constant can never select different branches. Case ranges are skipped until range overlap analysis is available.

## Configuration

```toml
[rules]
duplicate-switch-case = "error"
```

## Examples

### Bad

```pawn
const ONE = 1;

main()
{
    new value;
    switch (value)
    {
        case 1: return 1;
        case 2, ONE: return 2;
        case 3 - 2: return 3;
    }
}
```

### Good

```pawn
main()
{
    new value;
    switch (value)
    {
        case 1: return 1;
        case 2, 3: return 2;
        case 4 .. 6: return 3;
    }
}
```
