# invalid-shift-count

Reports constant shift counts outside the 32-bit cell width

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | constants, bitwise, control-flow |

## Details

Pawn cells are 32 bits wide. A constant shift count must be between 0 and 31.
