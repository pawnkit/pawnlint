# inconsistent-enum-prefix

Reports enum entries that omit a dominant member prefix

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | naming, style, enums |

## Details

A named enum with a dominant member prefix is easier to scan and less likely to contribute inconsistent global names. The rule uses the first underscore or case boundary and requires at least four definite entries. A prefix must appear on at least three entries and 75 percent of the enum.
