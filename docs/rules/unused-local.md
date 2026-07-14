# unused-local

Reports local variables that are never referenced

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | unused, variables, semantic |

## Details

A local variable that is never referenced adds noise and may indicate unfinished
code. Names beginning with `_` are treated as intentionally unused. The rule
does not offer a fix because an initializer may have side effects.
