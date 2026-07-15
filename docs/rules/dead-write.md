# dead-write

Reports local assignments whose stored value is never read

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | control-flow, assignments, data-flow |

## Details

An assignment is dead when every following path overwrites the local variable or exits before reading it. Only direct, standalone assignments with unambiguous control flow are checked.

## Configuration

```toml
[rules]
dead-write = "warning"
```

## Examples

### Bad

```pawn
overwritten()
{
    new value;
    value = 1;
    value = 2;
    Use(value);
}

before_exit()
{
    new value;
    value = 1;
    return;
}

branches(bool:condition)
{
    new value;
    if (condition)
        value = 1;
    else
        value = 2;
}
// …
```

### Good

```pawn
used_after_write(bool:condition)
{
    new value;
    value = 1;
    Use(value);

    if (condition)
        value = 2;
    else
        value = 3;
    Use(value);
}

other_symbols(parameter)
{
    global_value = 1;
    parameter = 2;
}

do_condition()
{
    new value;
    do
    {
        value = 1;
    }
    while (value);
}
// …
```
