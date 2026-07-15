# settimerex-format-argument-count

Reports SetTimerEx() calls whose specifier string and argument count differ

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | timer, format, arguments |

## Details

SetTimerEx's specifier string lists one letter per packed argument: i or d
(integer), f (float), s (string), b (boolean), a (array, immediately followed
by its own i size specifier). A mismatch between the specifiers and the
argument list passes the wrong values to the callback:

```pawn
SetTimerEx("OnDone", 1000, false, "dd", playerid); // "dd" needs 2 arguments
```

The rule only checks calls with a literal specifier string using the
documented letters; anything else is skipped rather than guessed at.

## Configuration

```toml
[rules]
settimerex-format-argument-count = "error"
```

## Examples

### Bad

```pawn
main()
{
    SetTimerEx("OnDone", 1000, false, "dd", 0);
    SetTimerEx("OnDone", 1000, false, "d", 0, 1);
    SetTimerEx("OnDone", 1000, false, "ai", myArray);
}
```

### Good

```pawn
forward OnDone(playerid, Float:amount);
public OnDone(playerid, Float:amount)
{
    return playerid + floatround(amount);
}

main()
{
    SetTimerEx("OnDone", 1000, false, "df", 0, 1.5);
    SetTimerEx("OnDone", 1000, false, "i", 0);
    SetTimerEx("OnDone", 1000, false, "s", "hello");
    SetTimerEx("OnDone", 1000, false, "b", true);

    new fmt[8] = "i";
    SetTimerEx("OnDone", 1000, false, fmt, 0);
}
```
