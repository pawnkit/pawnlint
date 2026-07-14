# float-equality

Reports Float values compared with == or !=

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | float, comparison, semantic |

## Details

Float values accumulate rounding error, so `==` and `!=` rarely mean
what they appear to:

```pawn
if (Float:distance == 0.0)
```

Use a tolerance comparison, or round both sides with `floatround` first if an
exact match is really intended:

```pawn
if (floatround(distance) == floatround(target))
```

The rule does not report when either side is a direct `floatround(...)` call,
since rounding first is the standard way to compare floats exactly. No fix is
offered because the correct tolerance or rounding style depends on context.
