# overwritten-copy

Reports memory copies overwritten before any access

| | |
| --- | --- |
| Category | performance |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | copies, buffers, memcpy, performance |

## Details

A memcpy has no useful effect when the next access to its local destination is an independent memcpy that covers the entire written byte range. The rule requires direct statements, one-dimensional local buffers, constant ranges, and the same lexical block. Partial, dynamic, branched, macro-derived, read, escaped, and self-copy cases are ignored.

## Configuration

```toml
[rules]
overwritten-copy = "warning"
```

## Examples

### Bad

```pawn
Check(const first[], const second[])
{
    new full[16];
    new ranged[16];
    memcpy(full, first, 0, 16 * 4);
    memcpy(full, second, 0, 16 * 4);
    Consume(full);
    memcpy(ranged[2], first, 0, 8);
    memcpy(ranged, second, 0, 16 * 4);
    Consume(ranged);
}
```

### Good

```pawn
Check(const first[], const second[], size)
{
    new read[16];
    new partial[16];
    new dynamic[16];
    new self[16];
    new branched[16];
    new controlled[16];
    memcpy(read, first, 0, 16 * 4);
    Consume(read);
    memcpy(read, second, 0, 16 * 4);
    memcpy(partial, first, 0, 16 * 4);
    memcpy(partial, second, 0, 8 * 4);
    memcpy(dynamic, first, 0, size);
    memcpy(dynamic, second, 0, 16 * 4);
    memcpy(self, first, 0, 16 * 4);
    memcpy(self, self, 0, 16 * 4);
    memcpy(branched, first, 0, 16 * 4);
    if (size > 0) {
        memcpy(branched, second, 0, 16 * 4);
    }
    memcpy(controlled, first, 0, 16 * 4);
    if (size > 0) {
        return;
    }
    memcpy(controlled, second, 0, 16 * 4);
}
```
