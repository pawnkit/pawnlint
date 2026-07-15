# unused-parameter

Reports unused parameters in non-public function definitions

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | unused, parameters, semantic |

## Details

An unused parameter may indicate dead code or an incomplete function. Public
and command-handler functions are skipped because external signatures may require every parameter.
Functions wrapped by a hooking library (`hook`, `inline`, and similar
single-word prefixes) are skipped for the same reason. Names beginning with
`_` or listed in a `#pragma unused` directive in the same function are
treated as intentionally unused.
