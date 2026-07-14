# format-argument-count

Reports literal format strings whose placeholders and arguments differ

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | format, arguments, native, api |

## Details

Pawn format placeholders consume variadic arguments in order. The rule checks direct calls to known formatted natives when the format is a literal and every specifier is recognized. Dynamic strings and named arguments are skipped.
