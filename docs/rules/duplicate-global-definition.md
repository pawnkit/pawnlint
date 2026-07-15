# duplicate-global-definition

Reports global variables defined more than once in one include graph

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | project |
| Default | enabled |
| Fixable | no |
| Tags | variables, project, includes |

## Details

A translation unit cannot contain multiple global variables with the same name. Separate entry-point files are checked independently.

## Configuration

```toml
[rules]
duplicate-global-definition = "error"
```

## Examples

### Bad

```pawn
new shared_value;
new shared_value;

main() {}
```

### Good

```pawn
new first_value;
new second_value;

main()
{
    first_value = second_value;
}
```
