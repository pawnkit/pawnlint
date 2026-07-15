# argument-tag-mismatch

Reports arguments incompatible with definite parameter tags

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | project |
| Default | enabled |
| Fixable | no |
| Tags | arguments, tags, calls, project, api |

## Details

Calls to resolved project functions and known APIs are checked for representation-changing Float mismatches and conflicts between distinct tags. Untagged non-Float values and zero are representation-compatible. Union, variadic, named, unresolved, macro-derived, uncertain, malformed, and structured-field arguments are ignored.

## Configuration

```toml
[rules]
argument-tag-mismatch = "warning"
```

## Examples

### Bad

```pawn
TeleportPlayer(playerid, Float:x, Float:y, Float:z)
{
    SetPlayerPos(playerid, x, y, z);
}

HandleTeleport(playerid, targetid)
{
    new Float:x, Float:y, Float:z;
    GetPlayerPos(targetid, x, y, z);
    TeleportPlayer(playerid, targetid, y, z);
}
```

### Good

```pawn
TeleportPlayer(playerid, Float:x, Float:y, Float:z)
{
    SetPlayerPos(playerid, x, y, z);
}

HandleTeleport(playerid, targetid)
{
    new Float:x, Float:y, Float:z;
    GetPlayerPos(targetid, x, y, z);
    TeleportPlayer(playerid, x, y, z);
}
```
