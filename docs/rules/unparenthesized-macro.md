# unparenthesized-macro

Reports function-like macros whose replacement list or parameters lack protective parentheses

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | syntax |
| Default | enabled |
| Fixable | yes |
| Tags | macros, preprocessor |

## Details

A function-like `#define` is expanded by simple text substitution. If a
parameter is used next to an operator without its own parentheses, or the
whole replacement list is not itself parenthesized, the operators at a call
site can silently change the computed result (the classic `#define SQUARE(x) x*x`
called as `SQUARE(a+b)` bug). The rule only inspects replacement lists the
parser could parse as a single expression or statement, and only flags
parameter references or replacement lists that are direct operands of an
operator; already-parenthesized, call-argument, and subscript positions are
left alone. The fix always wraps the exact reported span in parentheses, which
never changes what it evaluates to.

## Configuration

```toml
[rules]
unparenthesized-macro = "warning"
```

## Examples

### Bad

```pawn
#define SQUARE(x) x * x
#define DOUBLE(x) x + x
#define NEGATE(x) -x
#define TERNARY(x) x ? 1 : 0
#define INCREMENT(x) x++
#define MAX_HP 100 + 50
#define RETURN_SUM(x, y) return x + y

main()
{
	new value = SQUARE(1 + 2);
}
```

### Good

```pawn
#define SQUARE(%0) ((%0) * (%0))

main()
{
    return SQUARE(1 + 2);
}
```
