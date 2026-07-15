# redundant-else

Reports else branches after unconditional control transfer

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | yes |
| Tags | control-flow, branches, style |

## Details

An else is redundant when the preceding branch always returns, breaks, continues, or jumps away. Removing the else keeps the alternative's scope and comments intact. Uncertain and malformed branches are ignored.
