# legacy-include

Reports official SA-MP wrapper includes when targeting open.mp

| | |
| --- | --- |
| Category | openmp |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | include, migration, compatibility, api |

## Details

The official open.mp compatibility wrappers emit compiler warnings and direct users to include open.mp instead. The rule reports only the exact wrapper names shipped by the pinned API revision.
