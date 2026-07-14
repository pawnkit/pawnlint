# unused-function

Reports internal functions unused by any translation unit

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | unused, functions, project |

## Details

An unreferenced internal function may be dead code. Main, public, stock, callback, command-handler, state-qualified, operator, and underscore-prefixed functions are skipped. Translation units containing parser error nodes are skipped because references may be hidden.
