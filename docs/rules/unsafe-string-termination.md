# unsafe-string-termination

Reports raw copies used as strings without EOS termination

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | strings, termination, buffers, memcpy |

## Details

A raw memcpy does not append EOS. The rule reports array destinations later passed to a known string parameter in the same function without an intervening EOS assignment or terminating string write. Unresolved, macro-derived, uncertain, malformed, and non-string uses are ignored.
