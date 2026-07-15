# statement-macro-hazard

Reports statement macros unsafe in unbraced control flow

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | macros, statements, control-flow |

## Details

A function-like macro with multiple unwrapped statements, an embedded terminating semicolon, or an unmatched if can change surrounding control flow. The rule accepts single expressions, blocks, do-while wrappers, and complete if-else expansions. Uncertain, inactive, malformed, and declaration-generating macros are ignored.

## Configuration

```toml
[rules]
statement-macro-hazard = "warning"
```

## Examples

### Bad

```pawn
#define TWO(x) First(x); Second(x)
#define TERMINATED(x) Work(x);
#define CONDITIONAL(x) if (x) Work()

main()
{
    TWO(1);
}
```

### Good

```pawn
#define WRAPPED(x) do { First(x); Second(x); } while (0)
#define BLOCK(x) { First(x); Second(x); }
#define COMPLETE(x) if (x) First(); else Second()
#define CALL(x) Work(x)
#define VALUE(x) ((x) + 1)
#define EMPTY()
#define DECLARE(name) forward name(); public name()

main()
{
    WRAPPED(1);
}
```
