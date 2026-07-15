# loop-invariant-call

Reports pure calls repeated with unchanged arguments in loops

| | |
| --- | --- |
| Category | performance |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | loops, calls, performance, purity |

## Details

A pure call with unchanged arguments returns the same result on every iteration. The rule checks calls marked pure in API metadata and selected deterministic standard-library natives. Mutable arrays, globals, changed locals, unresolved calls, macros, uncertain loops, and strlen calls handled by the dedicated rule are ignored.
