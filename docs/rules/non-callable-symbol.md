# non-callable-symbol

Reports calls whose callee resolves to a variable, not a function

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | call, shadowing, semantic |

## Details

A local, parameter, or global variable is not callable. The most common cause
is a variable whose name shadows a native or user function:

```pawn
new time;
printf("%d", time());  // time is now the local cell, not the time() native
```

The compiler rejects this with "invalid function call, not a valid address".
Rename the variable or the call to resolve the ambiguity.
