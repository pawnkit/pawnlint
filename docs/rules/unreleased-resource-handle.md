# unreleased-resource-handle

Reports local resource handles that can reach function exit without release

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | resource, handle, database, file, control-flow |

## Details

A local initialized from a known resource creator must be released on every path. The rule follows definite scalar aliases and simple project wrappers, then stops when ownership becomes ambiguous.
