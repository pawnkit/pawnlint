# unimplemented-function

Reports legacy API calls intentionally not implemented by open.mp

| | |
| --- | --- |
| Category | openmp |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | migration, compatibility, api |

## Details

The official open.mp includes retain removed SA-MP functions as forward declarations so calls fail with a specific compiler error. The rule reports those direct calls and includes replacement guidance when available.
