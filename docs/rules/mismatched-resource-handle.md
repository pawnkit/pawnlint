# mismatched-resource-handle

Reports handles passed to the wrong resource releaser

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | resource, handle, database, file, tag |

## Details

File, database, and database-result handles have distinct release functions. The rule reports calls only when the argument has one definite incompatible resource tag.
