# duplicate-condition

Reports repeated pure conditions in an if and else-if chain

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | conditions, branches, semantic |

## Details

A repeated pure condition in an else-if chain can never become true after the first copy was false. Calls and other expressions with side effects are skipped.

## Configuration

```toml
[rules]
duplicate-condition = "warning"
```

## Examples

### Bad

```pawn
main()
{
    new value;
    if (value > 0)
    {
    }
    else if (value < 0)
    {
    }
    else if ((value > 0))
    {
    }
}
```

### Good

```pawn
main()
{
    new value;
    if (value == 1)
    {
    }
    else if (value == 2)
    {
    }

    if (Check())
    {
    }
    else if (Check())
    {
    }
}
```
