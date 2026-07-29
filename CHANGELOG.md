# Changelog

## 1.7.33 - 2026-07-29

### Performance

- Avoid copying shared tokens when expanding rebased syntax.

## 1.7.32 - 2026-07-29

### Performance

- Use lower-memory compact syntax expansion from pawn-parser.

## 1.7.31 - 2026-07-29

### Changed

- Limit syntax expansion reuse to rebased trivia edits.

## 1.7.30 - 2026-07-29

### Changed

- Reuse pawn-analysis syntax when linting editor buffers.

## 1.7.29 - 2026-07-29

### Performance

- Reuse compatible preprocessing in small and single-file projects.

## 1.7.28 - 2026-07-29

### Performance

- Rebase syntax after grammar-neutral whitespace edits.

## 1.7.27 - 2026-07-29

### Performance

- Reuse unchanged local symbols during compatible editor edits.

## 1.7.26 - 2026-07-29

### Performance

- Reuse unchanged syntax during compatible editor edits.

## 1.7.25 - 2026-07-29

### Performance

- Removed the incremental name cache after editor benchmarks showed a
  regression.

## 1.7.24 - 2026-07-29

### Performance

- Use cached name and call-arity checks during incremental analysis.

## 1.7.23 - 2026-07-29

### Changed

- Pin release actions and verify the tag against this changelog.

## 1.7.22 - 2026-07-29

### Changed

- Read real-project revisions from `pawn-corpus`.
- Added SP-RP to the scheduled real-project suite.

## 1.7.21 - 2026-07-29

### Performance

- Cache function parameter facts in the project semantic model.
- Avoid repeating local tag and constant queries after project resolution.
- Report the slowest rules in the opt-in real-project timing test.

## 1.7.20 - 2026-07-28

### Fixed

- Updated pawn-analysis for guarded include cycles and macro-replaced
  callables.

## 1.7.19 - 2026-07-28

### Performance

- Share callable-variant lookups across files in a single project unit.
- Reduced median warm SAFW editor linting from about 173 ms to 166 ms,
  allocations from about 79 MB to 72 MB, and allocation count from about
  320,000 to 308,000 on the reference machine.

## 1.7.18 - 2026-07-28

### Performance

- Sort reference lists without reflection.
- Read reference offsets directly from the syntax tree.
- Reduced median warm SAFW editor linting from about 177 ms to 173 ms and
  allocation count from about 323,000 to 320,000 on the reference machine.

## 1.7.17 - 2026-07-28

### Performance

- Use an unstable sort for reference lists whose ordering keys are complete.
- Reduced median warm SAFW editor linting from about 187 ms to 177 ms on the
  reference machine.

## 1.7.16 - 2026-07-28

### Performance

- Reuse bounded include resolutions while project roots and available sources
  stay unchanged.
- Clear cached resolutions through the existing file invalidation hook.
- Reduced warm SAFW editor linting from about 191 ms to about 173 ms,
  allocations from about 82 MB to 79 MB, and allocation count from about
  352,000 to 323,000 on the reference machine.

## 1.7.15 - 2026-07-28

### Performance

- Cache call ordering keys while building the call graph.
- Resolve enclosing functions through per-file ranges.
- Reduced warm SAFW editor linting from about 205 ms to about 191 ms on the
  reference machine.

## 1.7.14 - 2026-07-28

### Performance

- Reuse unresolved symbol lookups within each project unit.
- Skip sorting reference lists that are already ordered.
- Reduced warm SAFW editor linting from about 218 ms to about 205 ms and
  allocation count from about 399,000 to 352,000 on the reference machine.

## 1.7.13 - 2026-07-28

### Performance

- Reuse bounded filesystem probes across warm include resolution.
- Reduced warm SAFW editor linting from about 230 ms to about 205 ms,
  allocations from about 87 MB to 81 MB, and allocation count by about 40,000
  on the reference machine.

## 1.7.12 - 2026-07-28

### Performance

- Reuse the most recent define environment across adjacent include lookups.
- Reduced warm SAFW editor linting from about 250 ms to about 230 ms on the
  reference machine.

## 1.7.11 - 2026-07-28

### Performance

- Bound cached define contexts and analysis variants so old editor
  configurations do not accumulate for the lifetime of the language server.

## 1.7.10 - 2026-07-28

### Performance

- Reuse immutable define contexts through the warm analysis cache.
- Reduced warm SAFW editor linting from about 271 ms to about 255 ms and
  allocations from about 92 MB to 87 MB on the reference machine.

## 1.7.9 - 2026-07-28

### Performance

- Store one syntax identity per indexed reference.
- Reserve resolution maps from direct semantic reference counts.
- Reduced warm SAFW editor linting from about 289 ms to about 271 ms and
  allocations from about 96 MB to 92 MB on the reference machine.

## 1.7.8 - 2026-07-28

### Performance

- Avoid deduplication maps when resolving functions in a single project unit.
- Reserve reference index capacity from existing semantic data.
- Updated pawn-project to avoid rebuilding canonical paths.
- Reduced warm SAFW editor linting from about 306 ms to about 289 ms and
  allocations from about 125 MB to 96 MB on the reference machine.

## 1.7.7 - 2026-07-28

### Performance

- Use fixed-size hashes for define and analysis cache keys.
- Reduced warm SAFW editor linting from about 337 ms to about 306 ms and
  allocations from about 176 MB to 125 MB on the reference machine.

## 1.7.6 - 2026-07-28

### Fixed

- Use shared analysis buffers when resolving includes, including unsaved
  changes.

### Performance

- Cache include metadata for each project build.
- Reduced warm SAFW editor linting from about 353 ms to about 337 ms on the
  reference machine.

## 1.7.5 - 2026-07-28

### Fixed

- Normalize shared-analysis filenames before matching diagnostics on Windows.

## 1.7.4 - 2026-07-28

### Performance

- Sort call-graph edges once after direct and runtime calls are combined.
- Reduced warm SAFW editor linting from about 382 ms to about 365 ms on the
  reference machine.

## 1.7.3 - 2026-07-28

### Fixed

- Corrected the SAFW benchmark recorded for v1.7.2.

## 1.7.2 - 2026-07-28

### Performance

- Build call-graph edges from resolved references instead of resolving every
  call twice.
- Reduced warm SAFW editor linting from about 388 ms to about 382 ms on the
  reference machine.

## 1.7.1 - 2026-07-28

### Performance

- Reuse one content hash across parse, syntax, walk, and semantic caches.
- Reduced warm SAFW editor linting by about 11% on the reference machine.

## 1.7.0 - 2026-07-28

### Performance

- Prepare unchanged include parses in parallel from pawn-analysis tokens.
- Reduced cold SAFW workspace diagnostics by about 8% on the reference
  machine.

## 1.6.2 - 2026-07-28

### Performance

- Updated pawn-analysis for lower preprocessor allocation and faster cold
  project analysis.

## 1.6.1 - 2026-07-28

### Performance

- Updated pawn-analysis to avoid copying expanded parser tokens.

## 1.6.0 - 2026-07-28

### Changed

- Reused pawn-analysis results for missing includes and include cycles in
  editor runs.

## 1.5.0 - 2026-07-28

### Changed

- Reused pawn-analysis results for callability, argument tags, unreachable
  code, and missing return values in editor runs.

### Performance

- Reduced warm SAFW editor linting from 436-483 ms to about 388-401 ms on the
  reference machine.

## 1.4.1 - 2026-07-28

### Performance

- Reused define-aware syntax walks and semantic models for unchanged include
  chains.
- Reduced warm SAFW editor linting from about 670 ms to about 535 ms on the
  reference machine.

## 1.4.0 - 2026-07-28

### Added

- Added cancellable project builds and editor diagnostics.

### Changed

- Stopped lint rules and include traversal when an editor request is
  superseded.

## 1.3.4 - 2026-07-26

### Changed

- Updated analysis dependencies for declaration-level reuse tracking.

## 1.3.3 - 2026-07-26

### Changed

- Updated to pawn-analysis v0.5.2 to avoid unused tag-cache setup.

## 1.3.2 - 2026-07-26

### Changed

- Updated to pawn-analysis v0.5.1 for bounded tag-result caching.

## 1.3.1 - 2026-07-26

### Changed

- Updated to pawn-analysis v0.5.0 for reusable function tag checks.

## 1.3.0 - 2026-07-26

### Added

- Added cached function assignments and symbols to rule contexts.

### Performance

- Reduced `redundant-initialization` from about 90 ms to 2 ms on the San
  Fierro Faction Wars fixture.
- Reduced `dead-write` from about 83 ms to under 1 ms on the same fixture.

## 1.2.3 - 2026-07-26

### Changed

- Updated to pawn-analysis v0.4.3.

## 1.2.2 - 2026-07-26

### Changed

- Updated to pawn-analysis v0.4.2 to reduce preprocessing allocations.
- Grouped calls by their function or loop before running call-order and
  loop-invariance checks.
- Used indexed token ranges when comparing conditional branches.

### Performance

- Reduced `required-call-order` from about 468 ms to 5 ms on the San Fierro
  Faction Wars fixture.
- Reduced `loop-invariant-call` from about 117 ms to 2 ms on the same fixture.
- Reduced `identical-branches` from about 113 ms to under 1 ms.

## 1.2.1 - 2026-07-26

### Changed

- Kept resource wrapper inference on compact project syntax.
- Added direct alias checks for resource lifetime rules.
- Updated to pawn-analysis v0.4.1.

### Performance

- Reduced `read-after-release` from about 246 ms to under 2 ms on the
  San Fierro Faction Wars fixture without changing its diagnostics.

## 1.2.0 - 2026-07-26

### Added

- Added rule requirements and scopes to lint metadata.

### Changed

- The engine now builds semantic and control-flow models from the enabled
  rules' combined requirements.

## 1.1.13 - 2026-07-26

### Fixed

- Reverted prepared-tree expansion in editor diagnostics because it increased
  memory use. Root tokens remain shared with pawn-analysis.

## 1.1.12 - 2026-07-26

### Changed

- Editor diagnostics now expand pawn-analysis's prepared syntax instead of
  parsing the open buffer again.

## 1.1.11 - 2026-07-25

### Changed

- Added the repository support record with CI validation.

## 1.1.10 - 2026-07-25

### Changed

- The walk/semantic cache now builds each `#define` set's cache-key text
  once per environment instead of rebuilding it for every include that
  shares it.

## 1.1.9 - 2026-07-25

### Added

- `project.Options.RootTokens` lets a caller supply the entry file's already
  tokenized form, so `DiagnoseWithCache` can pass through the tokens
  pawn-analysis already computed instead of tokenizing the file again.

## 1.1.8 - 2026-07-25

### Added

- `DiagnoseWithCache` takes an optional, already-computed pawn-analysis
  result and reuses it instead of analyzing the file again for
  `pawn-analysis:sema/*` diagnostics.

## 1.1.7 - 2026-07-24

### Changed

- `DiagnoseWithCache` now also reuses each include's CST walk and semantic
  model when its content and active `#define` set haven't changed, instead
  of rebuilding them on every call.

## 1.1.6 - 2026-07-24

### Added

- Added `editor.DiagnoseWithCache`, which reuses a parse cache across calls
  so unchanged includes are not re-parsed on every call. `Diagnose` is
  unchanged and calls it with no cache.

### Changed

- Made `editor.Diagnose` request only the project features its enabled
  rules need, instead of always resolving runtime calls and function
  effects for every include.

## 1.1.5 - 2026-07-24

### Changed

- Updated the shared analysis dependency to v0.1.15, which removes duplicate
  work and a quadratic scan from large-file analysis.

## 1.1.4 - 2026-07-23

### Changed

- Project manifests now provide the include paths used by the CLI, analyzer,
  and editor API.

### Fixed

- Kept nested quoted includes relative to their entry directory.

## 1.1.3 - 2026-07-23

### Fixed

- Updated shared analysis so tag names are not reported as undefined symbols.

## 1.1.2 - 2026-07-23

### Changed

- Updated to the current Pawn analysis release.

## 1.1.1 - 2026-07-23

### Fixed

- Updated analysis for current macro and include syntax.

## 1.1.0 - 2026-07-22

### Added

- Added `--check-config` for validating configured paths, entries, and includes.

### Fixed

- Evaluate Pawn compiler constants in conditional directives.

## 1.0.10 - 2026-07-22

### Fixed

- Resolve angle-bracket includes through compiler include paths only.

## 1.0.9 - 2026-07-22

### Fixed

- Keep static functions local when checking include graphs.
- Ignore YSI iterator declarations when checking backing arrays for duplicate globals.
- Accept YSI variadic functions through updated analysis support.

## 1.0.8 - 2026-07-21

### Fixed

- Recognised command macros aliased from another function macro.

## 1.0.7 - 2026-07-21

### Fixed

- Accepted Pawn macro patterns used by YSI and other existing includes.

## 1.0.6 - 2026-07-21

### Fixed

- Reported each duplicate definition once across entry points.

## 1.0.5 - 2026-07-21

### Fixed

- Expanded macro-defined tags before checking argument compatibility.
- Resolved nested quoted includes from the entry file's directory.

## 1.0.4 - 2026-07-21

### Fixed

- Honoured active `#endinput` guards when checking duplicate declarations.
- Allowed signed numeric literals in object-like macros without extra parentheses.
- Accepted PawnPlus generic tags and declaration macros through updated parser support.
