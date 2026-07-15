# missing-return-value

Reports value-returning functions with paths that return no value

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | control-flow, returns, correctness |

## Details

Once a function returns a value, every reachable exit should return a value. The rule reports bare returns and paths that reach the end of such a function.

## Configuration

```toml
[rules]
missing-return-value = "warning"
```

## Examples

### Bad

```pawn
falls_through(bool:condition)
{
    if (condition)
        return 1;
}

bare_return(bool:condition)
{
    if (condition)
        return 1;
    return;
}

switch_path(value)
{
    switch (value)
    {
        case 1: return 1;
        default: result = 0;
    }
}
```

### Good

```pawn
all_paths(bool:condition)
{
    if (condition)
        return 1;
    return 0;
}

no_value_result()
{
    if (condition)
        return;
}

value_or_loop(bool:condition)
{
    if (condition)
        return 1;
    while (true)
    {
    }
}
```
