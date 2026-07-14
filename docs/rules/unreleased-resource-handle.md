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

A local initialized from a known file or SQLite resource creator must be released on every path before the function exits. Tracking stops conservatively when ownership escapes to user code or another value.
