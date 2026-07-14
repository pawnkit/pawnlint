# forward-signature-mismatch

Reports definitions that do not match their forward declaration

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | functions, forward, signature, semantic |

## Details

A function definition must match its forward declaration. The rule compares the signature parts exposed by the parser and reports only definite differences.
