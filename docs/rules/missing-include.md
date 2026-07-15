# missing-include

Reports required includes that cannot be resolved

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | project |
| Default | enabled |
| Fixable | no |
| Tags | includes, project, configuration |

## Details

Required includes must resolve through the source directory and configured include paths. Optional #tryinclude directives and uncertain paths are skipped.
