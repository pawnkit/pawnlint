# argument-tag-mismatch

Reports arguments incompatible with definite parameter tags

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | project |
| Default | enabled |
| Fixable | no |
| Tags | arguments, tags, calls, project, api |

## Details

Calls to resolved project functions and known APIs are checked for representation-changing Float mismatches and conflicts between distinct tags. Untagged non-Float values and zero are representation-compatible. Union, variadic, named, unresolved, macro-derived, uncertain, malformed, and structured-field arguments are ignored.

## Configuration

```toml
[rules]
argument-tag-mismatch = "warning"
```

## Examples

### Bad

```pawn
Use(Float:value, bool:flag, raw)
{
    return raw;
}

Check(Float:f, bool:b, raw)
{
    Use(raw, raw, f);
    Use(b, f, raw);
    Measure(raw, raw);
    Measure(f, b);
}
```

### Good

```pawn
Use(Float:value, bool:flag, raw)
{
    return raw;
}

Accept({Float,_}:value)
{
    return _:value;
}

Check(Float:f, bool:b, WEAPON:weapon, raw)
{
    new Float:structured[1][1];
    Use(f, b, raw);
    Accept(f);
    Accept(raw);
    Measure(f, weapon);
    Use(0, 0, 0);
    Measure(0, 0);
    Use(structured[0][0], b, raw);
}
```
