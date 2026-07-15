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

## Configuration

```toml
[rules]
constant-overflow = "warning"
```

## Examples

### Bad

```pawn
const MAX_VALUE = cellmax;

Check()
{
    new addition = 2147483647 + 1;
    new named = MAX_VALUE + 1;
    new subtraction = -2147483648 - 1;
    new multiplication = 50000 * 50000;
    new division = cellmin / -1;
    new negation = -cellmin;
    new literal = 4294967296;
    new hexadecimal = 0x100000000;
    new binary = 0b100000000000000000000000000000000;
    return addition + named + subtraction + multiplication + division + negation + literal + hexadecimal + binary;
}
```

### Good

```pawn
Check(value)
{
    new maximum = 2147483647 + 0;
    new minimum = -2147483648;
    new parenthesized_minimum = -(2147483648);
    new subtraction = -2147483648 + 1;
    new multiplication = 40000 * 40000;
    new hexadecimal = 0xFFFFFFFF;
    new high_bit = 0x80000000;
    new colour = 4278216843;
    new leading_zero = 08 + 09;
    new runtime = value + 2147483647;
    new Float:floating = 2147483647.0 + 1.0;
    new shifted = 0xFF << 24;
    return maximum + minimum + parenthesized_minimum + subtraction + multiplication + hexadecimal + high_bit + colour + leading_zero + runtime + floatround(floating) + shifted;
}
```
