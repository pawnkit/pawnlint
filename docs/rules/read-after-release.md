# read-after-release

Reports local resource handles used after release

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | resource, handle, lifetime, control-flow, api |

## Details

A local handle is invalid after release or ownership transfer. The rule follows definite scalar aliases and simple project wrappers, then stops when ownership becomes ambiguous.

## Configuration

```toml
[rules]
read-after-release = "error"
```

## Examples

### Bad

```pawn
native Resource:Acquire();
native Release(Resource:resource);
native Consume(Resource:resource);

main()
{
    new Resource:resource = Acquire();
    Release(resource);
    Consume(resource);
}
```

### Good

```pawn
native Resource:Acquire();
native Release(Resource:resource);
native Consume(Resource:resource);

main()
{
    new Resource:resource = Acquire();
    Consume(resource);
    Release(resource);
}
```
