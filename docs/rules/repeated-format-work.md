# repeated-format-work

Reports invariant formatting repeated before a buffer is used

| | |
| --- | --- |
| Category | performance |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | strings, formatting, loops, performance |

## Details

Formatting the same values into an untouched local buffer on every loop iteration repeats work when the buffer is only consumed after the loop. The rule checks direct format, Format, and strformat statements with invariant inputs and ignores mutable substitutions, conditional calls, macros, uncertain loops, shadowed natives, and buffers accessed in the loop.
