# Changelog

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
