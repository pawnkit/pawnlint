# Rule index

Generated from rule metadata. Do not edit by hand.

Rules are stable unless their page marks them as preview.

| ID | Category | Severity | Default | Fixable | Summary |
| --- | --- | --- | --- | --- | --- |
| [`ambiguous-include`](ambiguous-include.md) | portability | warning | on | no | Reports includes shadowed by another matching file |
| [`argument-value-range`](argument-value-range.md) | correctness | error | on | no | Reports constant arguments outside API parameter bounds |
| [`assignment-in-condition`](assignment-in-condition.md) | suspicious | warning | on | no | An assignment used as an if/while condition is often a typo for == |
| [`boolean-complexity`](boolean-complexity.md) | maintainability | warning | off | no | Reports boolean expressions with too many logical operators |
| [`boolean-name`](boolean-name.md) | style | warning | off | no | Reports boolean declarations without an allowed prefix |
| [`buffer-size`](buffer-size.md) | correctness | error | off | no | Reports native size arguments larger than a declared buffer |
| [`callback-signature`](callback-signature.md) | correctness | error | off | no | Reports public callbacks that do not match the target API |
| [`conflicting-include-symbol`](conflicting-include-symbol.md) | correctness | error | on | no | Reports namespace collisions contributed by included files |
| [`confusable-identifier`](confusable-identifier.md) | suspicious | warning | off | no | Reports visible declarations with visually confusable names |
| [`constant-condition`](constant-condition.md) | suspicious | warning | off | no | Reports if and ternary conditions with a constant result |
| [`cyclomatic-complexity`](cyclomatic-complexity.md) | maintainability | warning | off | no | Reports functions with too many independent control-flow paths |
| [`dead-write`](dead-write.md) | suspicious | warning | off | no | Reports local assignments whose stored value is never read |
| [`declaration-order`](declaration-order.md) | style | warning | off | no | Reports declarations outside the configured source order |
| [`deprecated-function`](deprecated-function.md) | openmp | warning | off | no | Reports deprecated compatibility functions in open.mp |
| [`deprecated-native`](deprecated-native.md) | openmp | warning | off | no | Reports calls to natives deprecated by the selected API |
| [`disallowed-name`](disallowed-name.md) | restriction | warning | off | no | Reports declarations denied by configured name policies |
| [`discarded-expression`](discarded-expression.md) | suspicious | warning | on | no | A standalone expression with no side effects does nothing |
| [`discarded-repeating-timer`](discarded-repeating-timer.md) | correctness | warning | off | no | Reports repeating SetTimer/SetTimerEx handles discarded before they can be killed |
| [`discarded-resource-handle`](discarded-resource-handle.md) | correctness | warning | off | no | Reports resource handles discarded before they can be released |
| [`division-by-zero`](division-by-zero.md) | correctness | error | on | no | Reports division or remainder by a constant zero |
| [`duplicate-condition`](duplicate-condition.md) | suspicious | warning | off | no | Reports repeated pure conditions in an if and else-if chain |
| [`duplicate-function-definition`](duplicate-function-definition.md) | correctness | error | on | no | Reports functions defined more than once in one include graph |
| [`duplicate-global-definition`](duplicate-global-definition.md) | correctness | error | on | no | Reports global variables defined more than once in one include graph |
| [`duplicate-include`](duplicate-include.md) | maintainability | warning | off | no | Reports a file included more than once from the same source file |
| [`duplicate-switch-case`](duplicate-switch-case.md) | correctness | error | on | no | Reports repeated constant values in one switch statement |
| [`empty-condition-body`](empty-condition-body.md) | correctness | error | on | yes | Accidental semicolon after an if/while/for condition makes the following block unconditional |
| [`float-equality`](float-equality.md) | suspicious | warning | off | no | Reports Float values compared with == or != |
| [`forbidden-include`](forbidden-include.md) | restriction | error | off | no | Reports includes denied by project policy |
| [`format-argument-count`](format-argument-count.md) | correctness | error | off | no | Reports literal format strings whose placeholders and arguments differ |
| [`format-argument-tag`](format-argument-tag.md) | correctness | error | on | no | Reports definite tag mismatches in formatted native calls |
| [`forward-signature-mismatch`](forward-signature-mismatch.md) | correctness | error | on | no | Reports definitions that do not match their forward declaration |
| [`function-length`](function-length.md) | maintainability | warning | off | no | Reports functions spanning too many source lines |
| [`identical-branches`](identical-branches.md) | suspicious | warning | off | no | Reports if and ternary branches with identical code |
| [`identifier-length`](identifier-length.md) | style | warning | off | no | Reports declarations outside configured name-length limits |
| [`ignored-return-value`](ignored-return-value.md) | correctness | warning | off | no | Reports discarded results from APIs marked must-use |
| [`impossible-comparison`](impossible-comparison.md) | correctness | warning | on | no | Reports comparisons that cannot produce both results |
| [`include-cycle`](include-cycle.md) | correctness | error | on | no | Reports cycles in the resolved include graph |
| [`include-layering`](include-layering.md) | restriction | error | off | no | Reports dependencies outside a source layer's allowlist |
| [`inconsistent-enum-prefix`](inconsistent-enum-prefix.md) | style | warning | off | no | Reports enum entries that omit a dominant member prefix |
| [`invalid-sentinel-comparison`](invalid-sentinel-comparison.md) | correctness | error | off | no | Reports a native's result compared against the wrong INVALID_* constant |
| [`invalid-shift-count`](invalid-shift-count.md) | correctness | error | on | no | Reports constant shift counts outside the 32-bit cell width |
| [`large-local-array`](large-local-array.md) | performance | warning | off | no | Reports large automatic arrays allocated on the Pawn stack |
| [`legacy-include`](legacy-include.md) | openmp | warning | off | no | Reports official SA-MP wrapper includes when targeting open.mp |
| [`magic-value`](magic-value.md) | maintainability | warning | off | no | Reports unexplained numeric and string literals |
| [`maximum-nesting`](maximum-nesting.md) | maintainability | warning | off | no | Reports functions with deeply nested control statements |
| [`mismatched-resource-handle`](mismatched-resource-handle.md) | correctness | error | off | no | Reports handles passed to the wrong resource releaser |
| [`missing-include`](missing-include.md) | correctness | error | on | no | Reports required includes that cannot be resolved |
| [`missing-return-value`](missing-return-value.md) | correctness | warning | on | no | Reports value-returning functions with paths that return no value |
| [`misspelled-callback`](misspelled-callback.md) | suspicious | warning | off | no | Reports public functions one edit away from a target callback |
| [`multiple-declarations`](multiple-declarations.md) | style | warning | off | no | Reports statements that declare multiple variables |
| [`naming-convention`](naming-convention.md) | style | warning | off | no | Reports declarations that violate configured naming policies |
| [`native-argument-count`](native-argument-count.md) | correctness | error | off | no | Reports calls with an impossible number of arguments for a known native |
| [`negative-or-zero-array-size`](negative-or-zero-array-size.md) | correctness | error | on | no | Reports array dimensions that evaluate to zero or less |
| [`non-callable-symbol`](non-callable-symbol.md) | correctness | error | on | no | Reports calls whose callee resolves to a variable, not a function |
| [`non-public-callback`](non-public-callback.md) | correctness | warning | off | no | Reports functions named exactly like a callback but missing the public qualifier |
| [`npath-complexity`](npath-complexity.md) | maintainability | warning | off | no | Reports functions with too many acyclic execution paths |
| [`out-of-bounds-constant-index`](out-of-bounds-constant-index.md) | correctness | error | on | no | Reports constant indexes outside a known array dimension |
| [`overwritten-resource-handle`](overwritten-resource-handle.md) | correctness | warning | off | no | Reports resource handles overwritten before any use or release |
| [`possibly-uninitialized`](possibly-uninitialized.md) | correctness | warning | off | no | Reports local variables read before an explicit assignment on every path |
| [`prefer-const`](prefer-const.md) | maintainability | warning | off | no | Reports initialized local variables that are never modified |
| [`public-documentation`](public-documentation.md) | style | warning | off | no | Reports selected functions without complete API documentation |
| [`raw-tick-subtraction`](raw-tick-subtraction.md) | correctness | warning | off | no | Reports GetTickCount() subtracted directly instead of through a wraparound-safe helper |
| [`read-after-release`](read-after-release.md) | correctness | error | on | no | Reports local resource handles used after release |
| [`recursive-call`](recursive-call.md) | suspicious | warning | off | no | Reports direct and mutual recursion in the project call graph |
| [`redundant-boolean-comparison`](redundant-boolean-comparison.md) | suspicious | warning | off | no | Reports boolean expressions compared with true or false |
| [`redundant-else`](redundant-else.md) | maintainability | warning | off | yes | Reports else branches after unconditional control transfer |
| [`redundant-forward`](redundant-forward.md) | maintainability | warning | off | no | Reports forward declarations that are not needed before a definition |
| [`redundant-initialization`](redundant-initialization.md) | suspicious | warning | off | no | Reports local initial values overwritten before any read |
| [`redundant-parentheses`](redundant-parentheses.md) | style | warning | off | yes | Reports expression parentheses that do not affect parsing |
| [`redundant-tag`](redundant-tag.md) | maintainability | warning | off | yes | Reports tag overrides that repeat an expression's known tag |
| [`repeated-strlen-in-loop`](repeated-strlen-in-loop.md) | performance | warning | off | no | Reports loop conditions that repeatedly scan an unchanged local string |
| [`required-call-order`](required-call-order.md) | correctness | error | off | no | Reports API calls missing a required earlier call |
| [`restricted-syntax`](restricted-syntax.md) | restriction | warning | off | no | Reports configured language and dependency restrictions |
| [`self-assignment`](self-assignment.md) | correctness | warning | on | yes | Reports assignments that store a symbol back into itself |
| [`settimerex-format-argument-count`](settimerex-format-argument-count.md) | correctness | error | off | no | Reports SetTimerEx() calls whose specifier string and argument count differ |
| [`shadowed-variable`](shadowed-variable.md) | maintainability | warning | off | no | Reports local declarations that hide an outer variable |
| [`sscanf-format-argument-count`](sscanf-format-argument-count.md) | correctness | error | off | no | Reports sscanf() calls whose format string and argument count differ |
| [`suppression-reason`](suppression-reason.md) | restriction | warning | off | no | Reports suppression directives without an adequate reason |
| [`suspicious-comma-expression`](suspicious-comma-expression.md) | suspicious | warning | on | no | The comma operator chains sub-expressions; it is rarely intended in statements or returns |
| [`suspicious-negation`](suspicious-negation.md) | suspicious | warning | on | no | '!' binds tighter than &/\|/^/==/!=, so !x & y is (!x) & y |
| [`swapped-arguments`](swapped-arguments.md) | suspicious | warning | off | no | Reports native arguments whose tags match each other's parameters |
| [`target-constant-availability`](target-constant-availability.md) | openmp | error | off | no | Reports open.mp-only constants when targeting SA-MP |
| [`target-native-availability`](target-native-availability.md) | openmp | error | off | no | Reports open.mp-only native calls when targeting SA-MP |
| [`todo-policy`](todo-policy.md) | restriction | warning | off | no | Reports task comments that violate configured metadata policy |
| [`too-many-globals`](too-many-globals.md) | maintainability | warning | off | no | Reports files with too many global variables |
| [`too-many-parameters`](too-many-parameters.md) | maintainability | warning | off | no | Reports functions with too many parameters |
| [`unescaped-sql-format`](unescaped-sql-format.md) | correctness | warning | off | no | Reports mysql_format calls using %s for a non-literal string argument |
| [`unimplemented-function`](unimplemented-function.md) | openmp | error | off | no | Reports legacy API calls intentionally not implemented by open.mp |
| [`unknown-suppression`](unknown-suppression.md) | maintainability | warning | on | no | Reports unknown, malformed, or unused pawnlint suppression directives |
| [`unparenthesized-macro`](unparenthesized-macro.md) | correctness | warning | on | yes | Reports function-like macros whose replacement list or parameters lack protective parentheses |
| [`unreachable-code`](unreachable-code.md) | correctness | warning | on | no | Reports statements that cannot be executed |
| [`unreleased-resource-handle`](unreleased-resource-handle.md) | correctness | warning | off | no | Reports local resource handles that can reach function exit without release |
| [`unused-function`](unused-function.md) | maintainability | warning | off | no | Reports internal functions unused by any translation unit |
| [`unused-global`](unused-global.md) | maintainability | warning | off | no | Reports global variables unused by any translation unit |
| [`unused-include`](unused-include.md) | maintainability | warning | off | no | Reports includes with no contribution to a complete build |
| [`unused-label`](unused-label.md) | maintainability | warning | off | yes | Reports labels that are not targeted by a goto statement |
| [`unused-local`](unused-local.md) | maintainability | warning | off | no | Reports local variables that are never referenced |
| [`unused-parameter`](unused-parameter.md) | maintainability | warning | off | no | Reports unused parameters in non-public function definitions |
