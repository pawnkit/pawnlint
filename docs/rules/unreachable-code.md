# unreachable-code

Reports statements that cannot be executed

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | control-flow, unreachable, correctness |

## Details

Code after an unconditional return, jump, or non-terminating loop cannot execute. The rule skips functions with malformed or uncertain control flow.
