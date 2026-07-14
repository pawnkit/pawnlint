# out-of-bounds-constant-index

Reports constant indexes outside a known array dimension

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | constants, arrays, control-flow |

## Details

A constant index must be between zero and the array size minus one. The rule checks direct indexing when both the symbol and first dimension are known.
