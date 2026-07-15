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

An unreferenced internal function may be dead code. Main, externally callable functions, resolved timer targets, state-qualified functions, operators, and underscore-prefixed functions are skipped. Translation units containing parser errors are skipped.
