# invariant-loop-condition

Reports loop conditions unchanged by their loop

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | loops, conditions, data-flow, semantic |

## Details

A condition based only on unchanged local scalars has the same result on every iteration. Conditions with calls, parameters, globals, arrays, macros, assignments, or uncertain references are ignored.

## Configuration

```toml
[rules]
invariant-loop-condition = "warning"
```

## Examples

### Bad

```pawn
Check()
{
    new remaining = 10;
    while (remaining > 0)
    {
        print("waiting");
    }

    new limit = 5;
    for (new index = 0; limit < 10; index++)
    {
        print("limited");
    }

    new ready;
    do
    {
        print("checking");
    }
    while (!ready);

    new lower = 1;
    new upper = 10;
    while (lower < upper)
    {
        break;
    }
}
```

### Good

```pawn
new global_running;

Check(parameter)
{
    new remaining = 10;
    while (remaining > 0)
        remaining--;

    for (new index = 0; index < 10; index++)
        print("working");

    for (new item = 0, length = 10; item != length; ++item)
        print("multiple");

    for (new reverse = 10; --reverse != -1;)
        print("condition update");

    while (IsReady())
        print("waiting");

    while (global_running)
        print("global");

    while (parameter)
        print("parameter");

    new changed = 1;
    while (changed)
        Mutate(changed);

// …
```
