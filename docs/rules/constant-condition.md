# constant-condition

Reports if and ternary conditions with a constant result

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | constants, conditions, control-flow |

## Details

A constant condition always selects the same branch. Loops are skipped because constant loop conditions are often intentional.

## Configuration

```toml
[rules]
constant-condition = "warning"
```

## Examples

### Bad

```pawn
main()
{
    if (1)
    {
    }
    if (2 - 2)
    {
    }
    new value = 1 ? 10 : 20;
}

propagated(bool:condition)
{
    new value;
    if (condition)
        value = 4;
    else
        value = 4;
    if (value == 4)
    {
    }
}
```

### Good

```pawn
main(value)
{
    if (value)
    {
    }
    while (1)
    {
        break;
    }

    if (value)
        value = 1;
    else
        value = 2;
    if (value == 1)
    {
    }
}
```
