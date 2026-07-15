# duplicate-include

Reports a file included more than once from the same source file

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | includes, project, dependencies |

## Details

Repeated directives that resolve to the same file are often redundant. The rule does not remove them because some Pawn includes intentionally support repeated expansion.
