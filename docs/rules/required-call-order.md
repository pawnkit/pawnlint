# required-call-order

Reports API calls missing a required earlier call

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | calls, order, api, contracts, control-flow |

## Details

API metadata can require other natives to be called earlier on every path through the same function. Calls in uncertain control flow, nested calls, and expressions without a definite evaluation order are skipped.
