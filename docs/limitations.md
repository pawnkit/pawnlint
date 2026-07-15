# Limitations

## Current analysis

- Syntax analysis
- Per-file semantic analysis
- Per-function control-flow analysis
- Basic include-graph analysis
- Preview interprocedural taint analysis for configured direct flows

## Not implemented yet

- Operator-overload and advanced tag compatibility analysis
- General interprocedural and alias-aware data-flow analysis
- Complete cross-file constant, tag, and state resolution
- Complete API constant and behavioral metadata

## Fixes

- A fix is offered only when a diagnostic carries a proven-safe edit; coverage
  is intentionally small.
- Exact pawn-parser recovery edits are available for supported syntax errors.
- `--fix` and `--fix-safe` apply the same safe-edit set.
- `--diff` previews changes without writing files.
- Stdin input supports `--diff` but cannot be modified in place.

## Analyzer API

- The analyzer API provides diagnostics and actions, not editor or LSP integration.
- Cancellation is checked between contexts and files, not during a parser or rule call.

## Preprocessor

- Macros are not expanded.
- Literal integer conditions and `defined(NAME)` checks are evaluated.
- Known names include configured symbols, compiler predefines, active defines,
  and definitions exported by earlier includes.
- Builds treat absent names as undefined; paths and variants treat them as uncertain.
- Rules skip inactive and uncertain branches.

## Semantic analysis

- Resolves unambiguous functions, globals, locals, parameters, named enums,
  enum entries, and labels — within a single file.
- Distinguishes value, function, label, tag, and member-name contexts.
- Constant evaluation covers integer expressions, declared constants, and
  standard enum sequences.
- Not evaluated: custom enum increments, macros, cross-file values.
- Direct state selectors are modeled; complex automaton selectors stay
  conservative.
- Tags propagate only when the result is definite.

## API analysis

- Native signatures are generated from pinned open.mp and SA-MP include
  revisions.
- Checks direct-call arity and known open.mp deprecations.
- Configured JSON metadata can add plugin callbacks, natives, constants,
  buffers, deprecations, resource ownership, and taint contracts.
- SA-MP targets: reports direct native calls and unresolved constant
  references declared only by open.mp modules. Constant values are not
  modeled.
- open.mp targets: reports direct calls to legacy functions explicitly marked
  unimplemented or broken-deprecated by the official includes.
- Format strings: literal strings are checked against documented placeholders.
  Dynamic formats, unknown extensions, plugin natives, and macro wrappers are
  not modeled.
- Buffer checks require a direct one-dimensional array, a known declared
  capacity, and an explicit constant size.

## Resource analysis

- Tracks direct local file, database, and query acquisitions across
  control-flow branches.
- Recognizes official release natives, released-handle use, and known
  non-owning native uses.
- Configured project functions can declare owned returns and borrowed or
  transferred scalar parameters when resolution is unambiguous.
- Tracking stops once ownership may escape through user code, returns,
  aliases, or reference parameters.

## Security analysis

- Preview taint tracking follows configured callback inputs, callable outputs,
  direct assignments, known buffer writers, and project parameters.
- Unknown calls, aliases, macros, ambiguous resolution, and unsupported
  transformations stop tracking.

## Project analysis

- Resolves direct includes from the source directory and configured include
  paths.
- Tracks define contexts and include order while sharing parsed CSTs.
- Resolves unambiguous function and value references; reports duplicate or
  unused project symbols.
- Cross-file constant evaluation is not available.
- Macro-generated includes are not expanded; uncertain includes are skipped.
- Contexts track macro names, not values; identical contexts share one instance.

## Naming policy

- Naming conventions apply only when configured and use the first matching
  selector.
- Disallowed-name policies apply only to configured exact names or patterns.
- Confusable names cover the Pawn ASCII groups `0/O/o` and `1/I/l`.
- Enum prefixes require four definite entries and 75 percent agreement.
- Identifier lengths apply only through configured ordered limits.
- Boolean naming applies only to declarations with one definite `bool` tag.
- Restricted calls and recursion require definite project resolution.
- Callbacks and natives require explicit opt-in. Uncertain declarations and
  non-standard identifiers are skipped.

## Control flow

- Tracks reachability, explicit assignment, and constant values for scalar
  non-static locals.
- Redundant initialization requires a pure scalar initializer and direct
  standalone overwrites.
- A value stays known only when every incoming path agrees.
- Calls invalidate local arguments when reference behavior is unknown.

## Dynamic calls

- The project call graph resolves only direct, unambiguous calls.
- Dynamic calls by name, function references, callbacks, timers, hooks, and
  generated dispatch remain conservative graph roots or unresolved edges.

## Pawn semantics

Pawn differs from C in tags, arrays, cells, and several operators. Validate
rules against actual Pawn and open.mp behavior — never assume C semantics.
