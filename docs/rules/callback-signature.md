# callback-signature

Reports public callbacks that do not match the target API

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | callbacks, openmp, samp, api |

## Details

Callback names and parameters are defined by the selected open.mp or SA-MP API. The rule checks public functions against checked-in API metadata.

## Configuration

```toml
[rules]
callback-signature = "error"
```

## Examples

### Bad

```pawn
public OnPlayerConnect(player)
{
    return 1;
}

public OnPlayerDeath(playerid, killerid, reason)
{
    return 1;
}
```

### Good

```pawn
public OnGameModeInit()
{
    return 1;
}

public OnPlayerConnect(playerid)
{
    return 1;
}

public OnCustomEvent(value)
{
    return value;
}
```
