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
#define LOG_BOTH(%0) print(%0); printf("Logged: %s", %0)

main()
{
    LOG_BOTH("server started");
}
```

### Good

```pawn
#define LOG_BOTH(%0) do { print(%0); printf("Logged: %s", %0); } while (0)

main()
{
    LOG_BOTH("server started");
}
```
