# Limitations

The incremental cache reuses rule diagnostics after rebuilding the project graph;
it does not cache parser or semantic models.

## Current analysis

- Syntax analysis
- Per-file semantics with project-wide fact resolution
- Per-function control-flow analysis
- Basic include-graph analysis
- Preview interprocedural taint analysis for configured sources and sinks

## Not implemented yet

- Operator-overload and advanced tag compatibility analysis
- General interprocedural data-flow outside ownership and taint
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

- Project files expose separate expanded CST and semantic views for active
  object-like and function-like macros.
- Nested macro tokens retain definition and invocation origins across includes.
- Stringizing, token pasting, conditional macro bodies, and parent macros used
  inside included files remain unexpanded.
- Literal integer conditions and `defined(NAME)` checks are evaluated.
- Known names include configured symbols, compiler predefines, active defines,
  and definitions exported by earlier includes.
- Builds treat absent names as undefined; paths and variants treat them as uncertain.
- Rules skip inactive and uncertain branches.
- Rules use expanded views only when their diagnostics can map safely to source.

## Semantic analysis

- Resolves unambiguous functions, globals, named enums, enum entries, and state
  variants across definite include units. Locals, parameters, and labels remain
  file-scoped.
- Distinguishes value, function, label, tag, and member-name contexts.
- Constant evaluation covers integer expressions, declared constants, and
  standard enum sequences.
- Not evaluated: custom enum increments and macros.
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
  transferred scalar parameters.
- Definite scalar aliases and simple project acquisition or transfer wrappers
  are tracked across files.
- Ambiguous calls, named arguments, complex wrappers, returns, and reference
  parameters remain ownership escapes.

## Security analysis

- Preview taint tracking follows configured sources through local expressions,
  buffer writers, project calls, returns, and output parameters.
- Globals, project array mutations, unknown calls, macros, ambiguous resolution,
  and unsupported transformations stop tracking.

## Project analysis

- Resolves direct includes from the source directory and configured include
  paths.
- Tracks define contexts and include order while sharing parsed CSTs.
- Resolves function and value references, constant values, tags, and compatible
  state variants; reports duplicate or unused project symbols.
- Function effects propagate definite global access, reference-parameter
  mutation, calls, and purity across resolved project functions.
- Effects are unknown for unresolved calls, macros, malformed syntax, ambiguous
  reference flows, and complex state selectors.
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
- Tracks definite scalar alias classes through direct copies and branch joins.
- Redundant initialization requires a pure scalar initializer and direct
  standalone overwrites.
- A value stays known only when every incoming path agrees.
- Calls invalidate exact mutated arguments when source, project, or API effects
  are known. Unknown calls invalidate direct scalar arguments.

## Dynamic calls

- The project call graph resolves only direct, unambiguous calls.
- Dynamic calls by name, function references, callbacks, timers, hooks, and
  generated dispatch remain conservative graph roots or unresolved edges.

## Pawn semantics

Pawn differs from C in tags, arrays, cells, and several operators. Validate
rules against actual Pawn and open.mp behavior — never assume C semantics.
