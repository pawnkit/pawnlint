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

A local handle returned by a native with release metadata is invalid after its matching releaser is called. The rule follows direct control-flow paths and stops at reassignment or ownership escape before release.
