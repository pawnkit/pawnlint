# Rule index

Generated from rule metadata. Do not edit by hand.

| ID | Category | Severity | Default | Fixable | Summary |
| --- | --- | --- | --- | --- | --- |
| [`assignment-in-condition`](assignment-in-condition.md) | suspicious | warning | on | no | An assignment used as an if/while condition is often a typo for == |
| [`buffer-size`](buffer-size.md) | correctness | error | off | no | Reports native size arguments larger than a declared buffer |
| [`callback-signature`](callback-signature.md) | correctness | error | off | no | Reports public callbacks that do not match the target API |
| [`comparison-chain`](comparison-chain.md) | suspicious | warning | on | no | Chained relational comparisons (a < b < c) do not test a range |
| [`constant-condition`](constant-condition.md) | suspicious | warning | off | no | Reports if and ternary conditions with a constant result |
| [`dead-write`](dead-write.md) | suspicious | warning | off | no | Reports local assignments whose stored value is never read |
| [`deprecated-function`](deprecated-function.md) | openmp | warning | off | no | Reports deprecated compatibility functions in open.mp |
| [`deprecated-native`](deprecated-native.md) | openmp | warning | off | no | Reports calls to natives deprecated by the selected API |
| [`discarded-expression`](discarded-expression.md) | suspicious | warning | on | no | A standalone expression with no side effects does nothing |
| [`discarded-resource-handle`](discarded-resource-handle.md) | correctness | warning | off | no | Reports resource handles discarded before they can be released |
| [`division-by-zero`](division-by-zero.md) | correctness | error | on | no | Reports division or remainder by a constant zero |
| [`duplicate-condition`](duplicate-condition.md) | suspicious | warning | off | no | Reports repeated pure conditions in an if and else-if chain |
| [`duplicate-function-definition`](duplicate-function-definition.md) | correctness | error | on | no | Reports functions defined more than once in one include graph |
| [`duplicate-global-definition`](duplicate-global-definition.md) | correctness | error | on | no | Reports global variables defined more than once in one include graph |
| [`duplicate-switch-case`](duplicate-switch-case.md) | correctness | error | on | no | Reports repeated constant values in one switch statement |
| [`empty-condition-body`](empty-condition-body.md) | correctness | error | on | yes | Accidental semicolon after an if/while/for condition makes the following block unconditional |
| [`format-argument-count`](format-argument-count.md) | correctness | error | off | no | Reports literal format strings whose placeholders and arguments differ |
| [`forward-signature-mismatch`](forward-signature-mismatch.md) | correctness | error | on | no | Reports definitions that do not match their forward declaration |
| [`identical-branches`](identical-branches.md) | suspicious | warning | off | no | Reports if and ternary branches with identical code |
| [`invalid-shift-count`](invalid-shift-count.md) | correctness | error | on | no | Reports constant shift counts outside the 32-bit cell width |
| [`large-local-array`](large-local-array.md) | performance | warning | off | no | Reports large automatic arrays allocated on the Pawn stack |
| [`legacy-include`](legacy-include.md) | openmp | warning | off | no | Reports official SA-MP wrapper includes when targeting open.mp |
| [`mismatched-resource-handle`](mismatched-resource-handle.md) | correctness | error | off | no | Reports handles passed to the wrong resource releaser |
| [`missing-return-value`](missing-return-value.md) | correctness | warning | on | no | Reports value-returning functions with paths that return no value |
| [`misspelled-callback`](misspelled-callback.md) | suspicious | warning | off | no | Reports public functions one edit away from a target callback |
| [`native-argument-count`](native-argument-count.md) | correctness | error | off | no | Reports calls with an impossible number of arguments for a known native |
| [`negative-or-zero-array-size`](negative-or-zero-array-size.md) | correctness | error | on | no | Reports array dimensions that evaluate to zero or less |
| [`out-of-bounds-constant-index`](out-of-bounds-constant-index.md) | correctness | error | on | no | Reports constant indexes outside a known array dimension |
| [`overwritten-resource-handle`](overwritten-resource-handle.md) | correctness | warning | off | no | Reports resource handles overwritten before any use or release |
| [`possibly-uninitialized`](possibly-uninitialized.md) | correctness | warning | off | no | Reports local variables read before an explicit assignment on every path |
| [`recursive-call`](recursive-call.md) | suspicious | warning | off | no | Reports direct and mutual recursion in the project call graph |
| [`redundant-boolean-comparison`](redundant-boolean-comparison.md) | suspicious | warning | off | no | Reports boolean expressions compared with true or false |
| [`repeated-strlen-in-loop`](repeated-strlen-in-loop.md) | performance | warning | off | no | Reports loop conditions that repeatedly scan an unchanged local string |
| [`self-assignment`](self-assignment.md) | correctness | warning | on | yes | Reports assignments that store a symbol back into itself |
| [`shadowed-variable`](shadowed-variable.md) | maintainability | warning | off | no | Reports local declarations that hide an outer variable |
| [`suspicious-comma-expression`](suspicious-comma-expression.md) | suspicious | warning | on | no | The comma operator chains sub-expressions; it is rarely intended in statements or returns |
| [`suspicious-negation`](suspicious-negation.md) | suspicious | warning | on | no | '!' binds tighter than &/\|/^/==/!=, so !x & y is (!x) & y |
| [`target-constant-availability`](target-constant-availability.md) | openmp | error | off | no | Reports open.mp-only constants when targeting SA-MP |
| [`target-native-availability`](target-native-availability.md) | openmp | error | off | no | Reports open.mp-only native calls when targeting SA-MP |
| [`unimplemented-function`](unimplemented-function.md) | openmp | error | off | no | Reports legacy API calls intentionally not implemented by open.mp |
| [`unknown-suppression`](unknown-suppression.md) | maintainability | warning | on | no | Reports unknown, malformed, or unused pawnlint suppression directives |
| [`unparenthesized-macro`](unparenthesized-macro.md) | correctness | warning | on | yes | Reports function-like macros whose replacement list or parameters lack protective parentheses |
| [`unreachable-code`](unreachable-code.md) | correctness | warning | on | no | Reports statements that cannot be executed |
| [`unreleased-resource-handle`](unreleased-resource-handle.md) | correctness | warning | off | no | Reports local resource handles that can reach function exit without release |
| [`unused-function`](unused-function.md) | maintainability | warning | off | no | Reports internal functions unused by any translation unit |
| [`unused-global`](unused-global.md) | maintainability | warning | off | no | Reports global variables unused by any translation unit |
| [`unused-label`](unused-label.md) | maintainability | warning | off | yes | Reports labels that are not targeted by a goto statement |
| [`unused-local`](unused-local.md) | maintainability | warning | off | no | Reports local variables that are never referenced |
| [`unused-parameter`](unused-parameter.md) | maintainability | warning | off | no | Reports unused parameters in non-public function definitions |
