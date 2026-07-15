# read-after-release

Reports local resource handles used after release

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | resource, handle, lifetime, control-flow, api |

## Details

A local handle is invalid after release or ownership transfer. The rule follows definite scalar aliases and simple project wrappers, then stops when ownership becomes ambiguous.
