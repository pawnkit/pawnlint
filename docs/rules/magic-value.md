# magic-value

Reports unexplained numeric and string literals

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | constants, literals, maintainability, policy |

## Details

Magic values hide the meaning of policy and domain constants. Named constants make reuse and changes safer. Declaration, array, and function-call exemptions keep generated data and fixed interfaces out of scope.

## Configuration

```toml
[rules]
magic-value = "warning"
```

Set options under `[rules.magic-value]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `check-numbers` | boolean | `true` | — | Check numeric literals |
| `check-strings` | boolean | `false` | — | Check string literals |
| `allowed-numbers` | string-list | `[-1 0 1 -1.0 0.0 1.0]` | — | Numeric values to allow |
| `allowed-strings` | string-list | `[]` | — | String contents to allow |
| `ignore-const-declarations` | boolean | `true` | — | Ignore literals in const declarations |
| `ignore-enums` | boolean | `true` | — | Ignore literals in enum declarations |
| `ignore-default-parameters` | boolean | `true` | — | Ignore parameter default values |
| `ignore-array-sizes` | boolean | `true` | — | Ignore array dimensions |
| `ignore-array-indexes` | boolean | `true` | — | Ignore array indexes |
| `ignore-array-literals` | boolean | `true` | — | Ignore values in array literals |
| `ignore-functions` | string-list | `[]` | — | Function name regular expressions whose arguments are ignored |

## Examples

### Bad

```pawn
Calculate(value)
{
    if (value > 42)
        return value * 60;

    SetTimer("Refresh", 2500, false);
    new Float:ratio = 3.5;

    switch (value)
    {
        case 7:
            return 1;
    }

    SendClientMessage(value, -1, "state");
    return 0;
}
```

### Good

```pawn
const MAX_RETRIES = 42;

enum State
{
    STATE_IDLE = 7,
    STATE_BUSY = 12
}

#define RETRY_DELAY (5000)

Check(value = 30)
{
    new values[16];
    new Float:zero = -0.0;
    values[2] = 100;
    CreatePoint(75, 125);
    SendClientMessage(value, -1, "ok");
    return value == 0 || value == 1 || value == -1 || value == 0xFFFFFFFF;
}
```
