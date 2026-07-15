# unconditional-recursion

Reports recursive cycles with no terminating path

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | project |
| Default | enabled |
| Fixable | no |
| Tags | recursion, calls, control-flow, project |

## Details

A recursive component cannot terminate when every reachable path in every member must call the component again. Base cases, conditional evaluation, non-recursive control-flow cycles, macros, unresolved calls, and uncertain functions suppress the diagnostic.
