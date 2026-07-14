# repeated-strlen-in-loop

Reports loop conditions that repeatedly scan an unchanged local string

| | |
| --- | --- |
| Category | performance |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | strings, loops, calls, performance |

## Details

A loop condition is evaluated on every iteration. Calling strlen there repeatedly scans the same local string when the loop neither writes it nor passes it to another call.
