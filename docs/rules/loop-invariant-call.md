# loop-invariant-call

Reports pure calls repeated with unchanged arguments in loops

| | |
| --- | --- |
| Category | performance |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | loops, calls, performance, purity |

## Details

A pure call with unchanged arguments returns the same result on every iteration. The rule checks inferred project-function effects, API purity metadata, and selected deterministic standard-library natives. Mutable arrays, globals, changed locals, unresolved calls, macros, uncertain loops, and strlen calls handled by the dedicated rule are ignored.

## Configuration

```toml
[rules]
loop-invariant-call = "warning"
```

## Examples

### Bad

```pawn
native PureNative(value);

PureFunction(value)
{
    return value + 1;
}

Check(limit)
{
    new total;
    for (new i; i < limit; i++) {
        total += PureNative(limit);
        total += PureFunction(4);
    }
    return total;
}
```

### Good

```pawn
native PureNative(value);
native MutableNative(value);

Check(limit, const text[])
{
    new total;
    for (new i; i < limit; i++) {
        total += PureNative(i);
        total += MutableNative(limit);
        limit--;
        total += floatabs(limit);
        total += strlen(text);
    }
    while ((limit = MutableNative(limit)) > 0) {
        total += PureNative(limit);
    }
    return total;
}
```
