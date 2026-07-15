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
GetSpawnWorld()
{
	new world;
	world = 1;
    world = 2;
    return world;
}
```

### Good

```pawn
GetSpawnWorld()
{
    new world = 1;
    if (random(2))
    {
        world = 2;
    }
    return world;
}
```
