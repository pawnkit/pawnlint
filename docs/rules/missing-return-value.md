# missing-return-value

Reports value-returning functions with paths that return no value

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | control-flow, returns, correctness |

## Details

Once a function returns a value, every reachable exit should return a value. The rule reports bare returns and paths that reach the end of such a function.
