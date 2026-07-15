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
