# discarded-expression

A standalone expression with no side effects does nothing

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | syntax |
| Default | enabled |
| Fixable | no |
| Tags | expression, dead-code |

## Details

A side-effect-free expression used as a statement does nothing:

```pawn
playerid + 1;
```

Calls, assignments, and updates are not reported. This rule has no fix because
the intended action is unknown.

## Configuration

```toml
[rules]
discarded-expression = "warning"
```

## Examples

### Bad

```pawn
main()
{
    playerid + 1;
    a * b;
    x && y;
    arr[0] | mask;
    (a + b);
}
```

### Good

```pawn
enum E_VALUES
{
    E_VALUE
}

main()
{
    Kick(playerid);
    x = 10;
    y++;
    z--;
    ++y;
    --z;
    foo(a, b, c);
    bar(x);
    return callcmd::goto(playerid, params);
}

StopTimers()
{
    stop timerid;
    stop timers[playerid];
}
```
