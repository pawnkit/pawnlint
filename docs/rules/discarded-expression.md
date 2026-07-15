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
    new score = 10;
    score + 5;
}
```

### Good

```pawn
main()
{
    new score = 10;
    score += 5;
    return score;
}
```
