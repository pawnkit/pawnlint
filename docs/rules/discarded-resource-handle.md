# discarded-resource-handle

Reports resource handles discarded before they can be released

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | resource, handle, database, file |

## Details

File, database, and database-result creators return handles that must be closed or freed. The rule reports direct standalone calls whose returned handle is immediately lost.
