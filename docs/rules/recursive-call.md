# recursive-call

Reports direct and mutual recursion in the project call graph

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | calls, recursion, project |

## Details

Recursive Pawn calls consume a fixed runtime stack and can overflow. The rule reports only statically resolved direct-call cycles and skips dynamic or ambiguous calls.
