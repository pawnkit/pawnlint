# redundant-tag

Reports tag overrides that repeat an expression's known tag

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | yes |
| Tags | tags, expressions, semantic |

## Details

A tag override is redundant when its operand already has exactly the same tag. Unknown, union, ambiguous, macro-dependent, and malformed expressions are ignored.
