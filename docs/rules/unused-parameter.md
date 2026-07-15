# unused-parameter

Reports unused parameters in non-public function definitions

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | unused, parameters, semantic |

## Details

An unused parameter may indicate dead code or an incomplete function. Public
and command-handler functions are skipped because external signatures may require every parameter.
Functions wrapped by a hooking library (`hook`, `inline`, and similar
single-word prefixes) are skipped for the same reason. Names beginning with
`_` or listed in a `#pragma unused` directive in the same function are
treated as intentionally unused.

## Configuration

```toml
[rules]
unused-parameter = "warning"
```

## Examples

### Bad

```pawn
stock Add(left, right)
{
    return left;
}

stock Empty(argc)
{
}

stock OtherPragmaScope(value)
{
}

stock UnrelatedPragma(value)
{
    #pragma unused OtherPragmaScope
}
```

### Good

```pawn
public OnPlayerConnect(playerid)
{
    return 1;
}

stock Add(left, right)
{
    return left + right;
}

stock Ignore(_value)
{
    return 1;
}

CMD:ExternalCommand(playerid, params[])
{
    return 1;
}

hook OnPlayerEnterVehicle(playerid, vehicleid, ispassenger)
{
    return playerid;
}

inline Response(pid, dialogid, response, listitem, string:inputtext[])
{
    return pid;
}
// …
```
