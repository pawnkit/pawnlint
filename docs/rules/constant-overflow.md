# constant-overflow

Reports constant arithmetic outside the cell range

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | constants, arithmetic, overflow, cells |

## Details

Pawn cells are 32-bit values. Definite integer addition, subtraction, multiplication, division, negation, and literals that lose bits are checked before cell wrapping. Floats, bitwise operations, runtime values, macros, and uncertain expressions are ignored.
