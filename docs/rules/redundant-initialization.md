# redundant-initialization

Reports local initial values overwritten before any read

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | control-flow, initialization, assignments, data-flow |

## Details

A pure scalar initializer is redundant when every following path overwrites the local or exits before reading its initial value. Static locals, loop declarations, side effects, uncertain flow, and non-standalone writes are skipped.

## Configuration

```toml
[rules]
redundant-initialization = "warning"
```

## Examples

### Bad

```pawn
GetWorld()
{
    new world = 0;
    world = 1;
    return world;
}
```

### Good

```pawn
GetWorld()
{
    new world = 1;
    return world;
}
```
