# declaration-order

Reports declarations outside the configured source order

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | declarations, order, style |

## Details

Top-level declaration groups follow the configured order. Omitted groups are ignored. Local variables can optionally be required before executable statements in each block. Inactive, uncertain, and malformed syntax is ignored.

## Configuration

```toml
[rules]
declaration-order = "warning"
```

Set options under `[rules.declaration-order]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `order` | string-list | `[include define enum constant variable native forward function]` | include, define, enum, constant, variable, native, forward, function | Top-level declaration group order |
| `locals-before-statements` | boolean | `false` | — | Require local declarations before executable statements |

## Examples

### Bad

```pawn
#include <core>

enum Values
{
    VALUE_FIRST
}

new globalValue;
const LATE_CONSTANT = 1;

main()
{
    print("start");
    new lateLocal;
    if (lateLocal)
    {
        print("nested");
        new nestedLate;
    }
    return lateLocal;
}
```

### Good

```pawn
#include <core>
#define VALUE (1)

enum Values
{
    VALUE_FIRST
}

const LIMIT = 10;
new globalValue;
native ExternalCall(value);
forward OnDeferred(value);

main()
{
    new localValue;
    localValue = ExternalCall(globalValue);
    return localValue;
}
```
