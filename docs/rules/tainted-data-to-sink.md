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

Configured sources are traced through local expressions, known buffer writers, project parameters, return values, and scalar reference outputs. The rule reports flows into configured sinks and stops when resolution or transformation is uncertain.
