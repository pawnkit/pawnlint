# forward-signature-mismatch

Reports definitions that do not match their forward declaration

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | functions, forward, signature, semantic |

## Details

A function definition must match its forward declaration. The rule compares the signature parts exposed by the parser and reports only definite differences.

## Configuration

```toml
[rules]
forward-signature-mismatch = "error"
```

## Examples

### Bad

```pawn
forward CountMismatch(value);

CountMismatch(value, extra)
{
}

forward Float:ReturnTag();

ReturnTag()
{
}

forward ParamTag(Float:value);

ParamTag(value)
{
}

forward ArrayRank(values[]);

ArrayRank(values)
{
}

forward Varargs(format[], ...);

Varargs(format[], value)
{
}
// …
```

### Good

```pawn
forward Float:Measure(const values[], count);

Float:Measure(const values[], count)
{
    return 0.0;
}

forward Log(format[], ...);

Log(format[], ...)
{
}
```
