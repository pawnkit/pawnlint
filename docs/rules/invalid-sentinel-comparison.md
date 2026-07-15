# invalid-sentinel-comparison

Reports a native's result compared against the wrong INVALID_* constant

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | native, constant, semantic, api |

## Details

open.mp and SA-MP use different sentinel values for different ID types
(`INVALID_PLAYER_ID`, `INVALID_VEHICLE_ID`, `INVALID_ACTOR_ID`,
`INVALID_OBJECT_ID`), and mixing them up is a common copy-paste mistake:

```pawn
new vehicleid = GetPlayerVehicleID(playerid);
if (vehicleid == INVALID_PLAYER_ID)
```

This always evaluates false for a vehicle ID, since a vehicle's invalid
sentinel is `INVALID_VEHICLE_ID`, not `INVALID_PLAYER_ID`. The rule checks a
curated set of well-known ID-returning natives against the sentinel constant
they should be compared with, and only reports when the compared name is
another known sentinel, not an unresolved or project-defined identifier.

## Configuration

```toml
[rules]
invalid-sentinel-comparison = "error"
```

## Examples

### Bad

```pawn
main()
{
    new vehicleid = GetPlayerVehicleID(0);
    if (vehicleid == INVALID_PLAYER_ID)
    {
    }
    new actorid = CreateActor(0, 0.0, 0.0, 0.0, 0.0);
    if (INVALID_OBJECT_ID != actorid)
    {
    }
}
```

### Good

```pawn
main()
{
    new vehicleid = GetPlayerVehicleID(0);
    if (vehicleid == INVALID_VEHICLE_ID)
    {
    }
    new actorid = CreateActor(0, 0.0, 0.0, 0.0, 0.0);
    if (actorid != INVALID_ACTOR_ID)
    {
    }
    new objectid = CreateObject(0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0);
    if (objectid == INVALID_OBJECT_ID)
    {
    }
    if (vehicleid == 0)
    {
    }
}
```
