# unreachable-code

Reports statements that cannot be executed

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | control-flow, unreachable, correctness |

## Details

Code after an unconditional return, jump, or non-terminating loop cannot execute. The rule skips functions with malformed or uncertain control flow.

## Configuration

```toml
[rules]
unreachable-code = "warning"
```

## Examples

### Bad

```pawn
after_return()
{
    return;
    new first;
    new second;
}

after_if()
{
    if (condition)
        return;
    else
        return;
    result = 1;
}

after_goto()
{
    goto done;
    skipped = 1;
done:
    return;
}
// …
```

### Good

```pawn
main()
{
    new value;
    if (value)
        return;
    value = 1;
}

with_jump()
{
    goto done;
done:
    return;
}

with_break()
{
    while (true)
    {
        break;
    }
    return;
}
```
