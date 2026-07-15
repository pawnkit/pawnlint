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
#define DOUBLE(x) ((x) + (x))
#define MAXIMUM(a, b) ((a) > (b) ? (a) : (b))
#define APPLY(%0) (Consume(%0) + Consume(%0))

main()
{
    new value = DOUBLE(NextValue());
}
```

### Good

```pawn
#define ADD_ONE(x) ((x) + 1)
#define ADD(a, b) ((a) + (b))
#define ARRAY_INFO(array) (sizeof(array) + (array)[0])
#define ARRAY_SHAPE(array) (sizeof(array) + tagof(array))
#define BUFFER(%0) format(%0, sizeof %0, "")
#define CONSTANT 5
#define NO_ARGS() (1 + 2)

main()
{
    new value = ADD_ONE(4);
}
```
