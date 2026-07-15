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
