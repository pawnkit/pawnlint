# Real-world validation

Milestone 8 was measured on 2026-07-15 against the pinned corpus revisions in `testdata/realworld/corpora.tsv`.

| Corpus | Files | Bytes | Tokens | Functions | Calls | Named calls | Timers | Findings |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| ScavengeSurvive `399ba8e` | 207 | 1,588,382 | 281,539 | 1,974 | 5,242 | 27 | 0 | 575 |
| gta-open `c7630a5` | 184 | 1,267,344 | 229,001 | 1,373 | 2,688 | 12 | 26 | 203 |
| AKRP-V5 `72aabd8` | 21 | 1,855,716 | 389,533 | 90 | 126 | 0 | 2 | 101 |

## Performance

Three warm process runs used the `all` profile. Runtime is wall-clock median; memory is median peak RSS.

| Corpus | Milestone 7 | Milestone 8 | Change | M7 memory | M8 memory | Change |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| ScavengeSurvive | 0.57 s | 0.60 s | +5% | 747 MiB | 539 MiB | -28% |
| gta-open | 0.34 s | 0.37 s | +9% | 474 MiB | 427 MiB | -10% |
| AKRP-V5 | 0.29 s | 0.28 s | -3% | 450 MiB | 339 MiB | -25% |

Pointer-based declaration identities removed repeated project-key allocation. Total ScavengeSurvive allocation fell from 1.30 GB to 626 MB. The contextual-include benchmark changed from 1.35 ms and 1.36 MB to 1.42 ms and 1.39 MB per build.

## Diagnostics

Repeated runs were deterministic. Review removed six false duplicate-definition diagnostics caused by comparing contextual instances of the same physical include.

Dependency snapshots are incomplete: ScavengeSurvive has 216 unresolved includes, gta-open 168, and AKRP-V5 57. Missing-include and declaration-order findings therefore dominate these totals.
