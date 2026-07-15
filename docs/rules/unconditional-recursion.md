# unconditional-recursion

Reports recursive cycles with no terminating path

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | project |
| Default | enabled |
| Fixable | no |
| Tags | recursion, calls, control-flow, project |

## Details

A recursive component cannot terminate when every reachable path in every member must call the component again. Base cases, conditional evaluation, non-recursive control-flow cycles, macros, unresolved calls, and uncertain functions suppress the diagnostic.

## Configuration

```toml
[rules]
unconditional-recursion = "warning"
```

## Examples

### Bad

```pawn
Direct()
{
    Direct();
}

ReturnDirect()
{
    return ReturnDirect();
}

BothBranches(value)
{
    if (value)
        BothBranches(value - 1);
    else
        BothBranches(value + 1);
}

MutualA()
{
    MutualB();
}

MutualB()
{
    MutualA();
}
// …
```

### Good

```pawn
WithBaseCase(value)
{
    if (value <= 0)
        return 0;
    return WithBaseCase(value - 1);
}

ConditionalCall(value)
{
    value && ConditionalCall(value - 1);
    return value;
}

MutualBaseA(value)
{
    if (value <= 0)
        return 0;
    return MutualBaseB(value - 1);
}

MutualBaseB(value)
{
    return MutualBaseA(value);
}
// …
```
