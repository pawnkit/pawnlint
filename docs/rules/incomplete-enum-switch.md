# incomplete-enum-switch

Reports enum switches that omit named values

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | switch, enums, coverage, project |

## Details

A switch over a resolved enum should cover every named value or provide a default clause. Enums with custom increments and switches with unknown cases, uncertain branches, ambiguous tags, or malformed syntax are ignored.

## Configuration

```toml
[rules]
incomplete-enum-switch = "warning"
```

## Examples

### Bad

```pawn
enum PlayerState
{
    STATE_NONE,
    STATE_ALIVE,
    STATE_DEAD
}

CheckState(PlayerState:current)
{
	switch (current)
	{
		case STATE_NONE:
			return 0;
	}
	return 1;
}
```

### Good

```pawn
enum PlayerState
{
    STATE_NONE,
    STATE_ALIVE,
    STATE_DEAD
}

IsAlive(PlayerState:state)
{
	switch (state)
	{
		case STATE_ALIVE: return 1;
		case STATE_NONE: return 0;
		case STATE_DEAD: return 0;
    }
    return 0;
}
```
