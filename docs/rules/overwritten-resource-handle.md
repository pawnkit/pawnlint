# overwritten-resource-handle

Reports resource handles overwritten before any use or release

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | resource, handle, database, file, control-flow |

## Details

Replacing a local file or SQLite handle loses the previous resource. The rule reports only two direct acquisitions connected by one linear control-flow path with no intervening reference.
