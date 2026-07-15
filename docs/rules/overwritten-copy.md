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
