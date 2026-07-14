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
Names beginning with `_` are treated as intentionally unused.
