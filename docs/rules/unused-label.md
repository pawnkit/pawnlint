# unused-label

Reports labels that are not targeted by a goto statement

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | yes |
| Tags | unused, labels, semantic |

## Details

A label with no matching `goto` target has no effect. The rule reports only
labels that resolve unambiguously within one function. The safe fix removes the
label and keeps the following statement unchanged.
