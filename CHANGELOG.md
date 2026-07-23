# Changelog

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
