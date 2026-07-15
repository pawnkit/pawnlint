# format-argument-tag

Reports definite tag mismatches in formatted native calls

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | format, arguments, native, api, tags |

## Details

The rule checks literal formats used by natives with formatParameter metadata. %f requires Float values, while integer specifiers reject Float values. String and library-dependent specifiers are skipped.
