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

## Configuration

```toml
[rules]
non-callable-symbol = "error"
```

## Examples

### Bad

```pawn
native time();

ShadowedNative()
{
    new time;
    time = 5;
    printf("%d", time());
}

CalledParameter(count)
{
    return count();
}

CalledGlobal()
{
    return gTotal();
}

new gTotal;
```

### Good

```pawn
native time();

main()
{
    printf("%d", time());

    new value = Add(1, 2);
    printf("%d", value);
}

Add(a, b)
{
    return a + b;
}

UnusedShadow()
{
    // A local sharing a native's name is fine as long as it is never
    // called; only calling it through this name is an error.
    new time;
    time = 5;
    printf("%d", time);
}
```
