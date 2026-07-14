# shadowed-variable

Reports local declarations that hide an outer variable

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | variables, shadowing, semantic |

## Details

A local variable with the same name as an outer variable can make code hard to
follow. The rule reports only unambiguous bindings and does not offer a rename
because related references may span more code.
