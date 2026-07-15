# misspelled-callback

Reports public functions one edit away from a target callback

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | callbacks, openmp, samp, api |

## Details

A one-character callback typo creates an ordinary public function that the server never calls. Only unique one-edit matches are reported.

## Configuration

```toml
[rules]
misspelled-callback = "warning"
```

## Examples

### Bad

```pawn
public OnPlayerConect(playerid)
{
    return 1;
}

public OnGameModeInt()
{
    return 1;
}
```

### Good

```pawn
public OnPlayerConnect(playerid)
{
    return 1;
}

public OnCustomEvent(value)
{
    return value;
}
```
