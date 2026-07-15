# format-argument-tag

Reports definite tag mismatches in formatted native calls

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | format, arguments, native, api, tags |

## Details

The rule checks literal formats used by natives with formatParameter metadata. %f requires Float values, while integer specifiers reject Float values. String and library-dependent specifiers are skipped.

## Configuration

```toml
[rules]
format-argument-tag = "error"
```

## Examples

### Bad

```pawn
#pragma rational Float

native LogValues(const format[], {Float, _}:...);

main() {
    new Float:ratio;
    new count;
    LogValues("%f %d", count, ratio);
    LogValues("%d", 1.0);
}
```

### Good

```pawn
#pragma rational Float

native LogValues(const format[], {Float, Custom, _}:...);

main() {
    new Float:ratio;
    new count;
    new Custom:identifier;
    LogValues("%f %d %x", ratio, count, identifier);
    LogValues("%f", 1.0);
}
```
