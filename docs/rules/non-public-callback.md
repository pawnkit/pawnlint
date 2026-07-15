# non-public-callback

Reports functions named exactly like a callback but missing the public qualifier

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | callbacks, openmp, samp, api |

## Details

The server dispatches callbacks by looking up a `public` function with the
exact callback name; a same-named function without `public` compiles cleanly
but is never called:

```pawn
OnPlayerConnect(playerid)
{
    // never runs; the server calls the public symbol, which does not exist
}
```

The rule reports functions whose name is an exact, case-sensitive match for a
known callback but that lack the `public` qualifier. Functions wrapped by a
hooking library (`hook OnPlayerConnect(...)`, y_hooks' convention) are skipped,
since the wrapper dispatches the callback itself. No fix is offered because a
same-named private helper may be intentional.

## Configuration

```toml
[rules]
non-public-callback = "warning"
```

## Examples

### Bad

```pawn
OnPlayerConnect(playerid)
{
    return 1;
}

stock OnPlayerDeath(playerid, killerid, reason)
{
    return 1;
}
```

### Good

```pawn
public OnPlayerConnect(playerid)
{
    return 1;
}

stock HelperFunction(playerid)
{
    return playerid;
}

static UpdateScore(playerid)
{
    return playerid;
}

hook OnPlayerDisconnect(playerid, reason)
{
    return 1;
}
```
