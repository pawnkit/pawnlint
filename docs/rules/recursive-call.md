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

Recursive Pawn calls consume a fixed runtime stack and can overflow. The rule reports statically resolved direct and named-call cycles and skips ambiguous targets.
