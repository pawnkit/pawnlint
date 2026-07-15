# infinite-loop

Reports loops proven unable to exit

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | loops, control-flow, conditions, termination |

## Details

A loop is infinite when its condition is definitely true, its condition values are unchanged, and no reachable break or return exits it. Gotos, macros, uncertain branches, calls in conditions, and unknown values suppress the diagnostic.

## Configuration

```toml
[rules]
infinite-loop = "warning"
```

## Examples

### Bad

```pawn
Forever()
{
    while (1)
    {
        print("forever");
    }
}

EmptyFor()
{
    for (;;)
    {
        print("forever");
    }
}

Invariant()
{
    new running = 1;
    while (running)
    {
        print("forever");
    }
}
// …
```

### Good

```pawn
BreakLoop()
{
    while (1)
    {
        break;
    }
}

ReturnLoop()
{
    for (;;)
    {
        return 1;
    }
}

ChangedCondition()
{
    new running = 1;
    while (running)
    {
        running = 0;
    }
}
// …
```
