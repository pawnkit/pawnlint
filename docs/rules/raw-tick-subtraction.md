# raw-tick-subtraction

Reports GetTickCount() subtracted directly instead of through a wraparound-safe helper

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | timing, overflow, samp, openmp |

## Details

GetTickCount() returns a 32-bit millisecond counter that wraps around after
about 24.8 days of server uptime. Subtracting two tick values directly breaks
on that wraparound:

```pawn
if (GetTickCount() - lastAction > 1000) // wrong after ~24.8 days uptime
```

Use a wraparound-safe difference helper (such as sscanf's bundled
GetTickCountDifference, or open.mp's Time_GetTickDifference) instead.

## Configuration

```toml
[rules]
raw-tick-subtraction = "warning"
```

## Examples

### Bad

```pawn
CheckCooldown(playerid)
{
    if (GetTickCount() - lastAction[playerid] > 1000)
    {
        return 1;
    }
    return 0;
}

new lastAction[500];

StoreDuration(started)
{
    new duration = GetTickCount() - started;
    return duration;
}
```

### Good

```pawn
Update(playerid)
{
    new elapsed = GetTickCountDifference(GetTickCount(), lastAction[playerid]);
    return elapsed;
}

new lastAction[500];

StorePastTimestamp()
{
    new due = GetTickCount() + 1000;
    return due;
}

forward GetTickCountDifference(a, b);
```
