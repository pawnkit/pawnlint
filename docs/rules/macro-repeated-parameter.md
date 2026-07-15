# macro-repeated-parameter

Reports macro parameters evaluated more than once

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | macros, evaluation, side-effects |

## Details

A function-like macro that evaluates one parameter more than once can repeat calls, assignments, and increments supplied by the caller. The rule checks fully parsed replacement lists and ignores unevaluated sizeof, tagof, and defined operands, opaque bodies, uncertain definitions, and malformed macros.

## Configuration

```toml
[rules]
macro-repeated-parameter = "warning"
```

## Examples

### Bad

```pawn
#define DOUBLE(%0) ((%0) + (%0))

main()
{
    new value = DOUBLE(random(10));
    return value;
}
```

### Good

```pawn
#define DOUBLE(%0) ((%0) * 2)

main()
{
    new value = DOUBLE(random(10));
    return value;
}
```
