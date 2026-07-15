# division-by-zero

Reports division or remainder by a constant zero

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | constants, arithmetic, control-flow |

## Details

Division and remainder by zero are invalid. The rule reports only operands that can be evaluated with certainty.

## Configuration

```toml
[rules]
division-by-zero = "error"
```

## Examples

### Bad

```pawn
const ZERO = 1 - 1;

main()
{
    new first = 10 / 0;
    new second = 10 % (3 - 3);
    first /= 0x0;
    second %= ZERO;

    new divisor = 0;
    first = 10 / divisor;
}
```

### Good

```pawn
Average(total, count)
{
    if (count == 0)
    {
        return 0;
    }
    return total / count;
}
```
