# redundant-parentheses

Reports expression parentheses that do not affect parsing

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | yes |
| Tags | expressions, parentheses, style |

## Details

Parentheses are redundant when removing them preserves Pawn precedence, associativity, argument boundaries, statement syntax, and assignment-condition intent. Macro, uncertain, and malformed syntax is ignored.
