# possibly-uninitialized

Reports local variables read before an explicit assignment on every path

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | control-flow, initialization, data-flow |

## Details

Pawn zero-fills local cells, but the compiler still tracks whether a local received an explicit value. This rule reports reads that can occur before an initializer or assignment. Resolved function effects distinguish read-only and mutating arguments. Unknown reference arguments stop tracking, while API parameters marked as outputs establish assignment.

## Configuration

```toml
[rules]
possibly-uninitialized = "warning"
```

## Examples

### Bad

```pawn
Use(value)
{
    return value;
}

direct_read()
{
    new value;
    Use(value);
}

one_branch(bool:condition)
{
    new value;
    if (condition)
        value = 1;
    Use(value);
}

optional_loop()
{
    new value;
    while (Check())
    {
        value = 1;
    }
    Use(value);
}
// …
```

### Good

```pawn
Use(value)
{
    return value;
}

SetValue(&value)
{
    value = 1;
}

initialized()
{
    new value = 1;
    Use(value);
}

both_branches(bool:condition)
{
    new value;
    if (condition)
        value = 1;
    else
        value = 2;
    Use(value);
}
// …
```
