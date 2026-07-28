# Changelog

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
