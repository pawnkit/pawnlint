# redundant-boolean-comparison

Reports boolean expressions compared with true or false

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | boolean, comparison, semantic |

## Details

A boolean expression does not need to be compared with a boolean literal. Use the expression directly or negate it.

## Configuration

```toml
[rules]
redundant-boolean-comparison = "warning"
```

## Examples

### Bad

```pawn
bool:IsReady()
{
    return true;
}

main()
{
    new bool:flag;
    if (flag == true)
    {
    }
    if (flag != false)
    {
    }
    if (false == flag)
    {
    }
    if (true != flag)
    {
    }
    if (IsReady() == false)
    {
    }
}
```

### Good

```pawn
main()
{
    new bool:flag;
    new value;
    if (flag)
    {
    }
    if (!flag)
    {
    }
    if (value == 1)
    {
    }
    if (value == true)
    {
    }
}
```
