# duplicate-function-definition

Reports functions defined more than once in one include graph

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | project |
| Default | enabled |
| Fixable | no |
| Tags | functions, project, includes |

## Details

A translation unit cannot contain multiple definitions of the same function. Separate entry-point files are checked independently.

## Configuration

```toml
[rules]
duplicate-function-definition = "error"
```

## Examples

### Bad

```pawn
Shared() {}
Shared() {}
```

### Good

```pawn
Start() {}
Stop() {}
```
