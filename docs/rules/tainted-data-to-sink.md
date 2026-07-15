# tainted-data-to-sink

Reports configured input reaching a configured sensitive sink

| | |
| --- | --- |
| Category | security |
| Severity | warning |
| Analysis | project |
| Stability | preview |
| Default | disabled |
| Fixable | no |
| Tags | security, taint, input, sink, project |

## Details

Configured callback inputs and callable output parameters are traced through direct local assignments, known buffer writers, and project function parameters. A diagnostic is reported when that data reaches a configured SQL, command, file, format, or custom sink. Unknown calls, unsupported transformations, ambiguous resolution, macros, and uncertain functions terminate the proof.
