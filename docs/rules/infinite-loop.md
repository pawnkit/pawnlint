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
WaitForPlayers()
{
    while (true)
    {
        print("waiting");
    }
}
```

### Good

```pawn
WaitForPlayers(attempts)
{
    while (attempts > 0)
    {
        attempts--;
    }
    return attempts;
}
```
