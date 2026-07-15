# too-many-parameters

Reports functions with too many parameters

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | size, functions, parameters, maintainability |

## Details

Named and variadic parameters count toward the configured maximum. Public functions and known callbacks are skipped by default because their signatures may be externally fixed. Name exclusions support project-specific interfaces.

## Configuration

```toml
[rules]
too-many-parameters = "warning"
```

Set options under `[rules.too-many-parameters]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `7` | minimum 1; maximum 1000 | Maximum parameters per function |
| `include-public` | boolean | `false` | — | Check public function signatures |
| `include-callbacks` | boolean | `false` | — | Check known callback signatures |
| `exclude` | string-list | `[]` | — | Function name regular expressions to exclude |

## Examples

### Bad

```pawn
TooMany(first, second, third, fourth)
{
    return first + second + third + fourth;
}

IncludesVariadic(first, second, third, ...)
{
    return first + second + third;
}
```

### Good

```pawn
Allowed(first, second, third)
{
    return first + second + third;
}

public OnExternalCallback(first, second, third, fourth, fifth)
{
    return first + second + third + fourth + fifth;
}

OnPlayerWeaponShot(playerid, weaponid, hittype, hitid, Float:x, Float:y, Float:z)
{
    return playerid + weaponid + hittype + hitid + floatround(x + y + z);
}

Generated_Interface(first, second, third, fourth, fifth)
{
    return first + second + third + fourth + fifth;
}

Variadic(const format[], ...)
{
    return format[0];
}
```
