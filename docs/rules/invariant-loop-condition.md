# invariant-loop-condition

Reports loop conditions unchanged by their loop

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | loops, conditions, data-flow, semantic |

## Details

A condition based only on unchanged local scalars has the same result on every iteration. Conditions with calls, parameters, globals, arrays, macros, assignments, or uncertain references are ignored.
