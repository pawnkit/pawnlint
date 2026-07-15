# ignored-return-value

Reports discarded results from APIs marked must-use

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | calls, return-value, api, contracts |

## Details

A native marked mustUse in API metadata requires callers to consume its result. Direct standalone calls are reported; nested, uncertain, macro-defined, and locally overridden calls are skipped.

## Configuration

```toml
[rules]
ignored-return-value = "warning"
```

## Examples

### Bad

```pawn
main() {
    RequiredResult();
}
```

### Good

```pawn
main() {
    new value = RequiredResult();
    OrdinaryResult();
}
```
