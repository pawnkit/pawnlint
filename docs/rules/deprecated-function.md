# deprecated-function

Reports deprecated compatibility functions in open.mp

| | |
| --- | --- |
| Category | openmp |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | deprecated, migration, compatibility, api |

## Details

Some legacy SA-MP APIs remain available as compatibility stocks but are documented as broken by the official open.mp includes. The rule reports direct calls and includes the official guidance.

## Configuration

```toml
[rules]
deprecated-function = "warning"
```

## Examples

### Bad

```pawn
main()
{
	new players = GetPlayerPoolSize();
	new vehicles = GetVehiclePoolSize();
	new actors = GetActorPoolSize();
	return players + vehicles + actors;
}
```

### Good

```pawn
main()
{
	return GetMaxPlayers();
}
```
