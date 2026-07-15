# unimplemented-function

Reports legacy API calls intentionally not implemented by open.mp

| | |
| --- | --- |
| Category | openmp |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | migration, compatibility, api |

## Details

The official open.mp includes retain removed SA-MP functions as forward declarations so calls fail with a specific compiler error. The rule reports those direct calls and includes replacement guidance when available.

## Configuration

```toml
[rules]
unimplemented-function = "error"
```

## Examples

### Bad

```pawn
main()
{
	EnableTirePopping(false);
	SetDeathDropAmount(100);
	SetTeamCount(4);
}
```

### Good

```pawn
main()
{
	SetPlayerTeam(0, 1);
	return GetMaxPlayers();
}
```
