package rules

import (
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules/correctness"
	"github.com/pawnkit/pawnlint/pkg/rules/maintainability"
	"github.com/pawnkit/pawnlint/pkg/rules/openmp"
	"github.com/pawnkit/pawnlint/pkg/rules/performance"
	"github.com/pawnkit/pawnlint/pkg/rules/suspicious"
)

func Register(reg *lint.Registrar) {
	reg.MustRegister(correctness.EmptyConditionBody{})
	reg.MustRegister(correctness.AssignmentInCondition{})
	reg.MustRegister(correctness.DiscardedExpression{})
	reg.MustRegister(suspicious.SuspiciousCommaExpression{})
	reg.MustRegister(suspicious.SuspiciousNegation{})
	reg.MustRegister(correctness.UnknownSuppression{})
	reg.MustRegister(correctness.SelfAssignment{})
	reg.MustRegister(correctness.UnparenthesizedMacro{})
	reg.MustRegister(maintainability.UnusedLocal{})
	reg.MustRegister(maintainability.UnusedParameter{})
	reg.MustRegister(maintainability.ShadowedVariable{})
	reg.MustRegister(maintainability.UnusedLabel{})
	reg.MustRegister(correctness.DivisionByZero{})
	reg.MustRegister(correctness.InvalidShiftCount{})
	reg.MustRegister(correctness.InvalidArraySize{})
	reg.MustRegister(correctness.ConstantCondition{})
	reg.MustRegister(correctness.DuplicateSwitchCase{})
	reg.MustRegister(correctness.OutOfBoundsConstantIndex{})
	reg.MustRegister(suspicious.DuplicateCondition{})
	reg.MustRegister(suspicious.RedundantBooleanComparison{})
	reg.MustRegister(correctness.ForwardSignatureMismatch{})
	reg.MustRegister(openmp.CallbackSignature{})
	reg.MustRegister(openmp.MisspelledCallback{})
	reg.MustRegister(openmp.TargetNativeAvailability{})
	reg.MustRegister(openmp.TargetConstantAvailability{})
	reg.MustRegister(openmp.UnimplementedFunction{})
	reg.MustRegister(openmp.DeprecatedFunction{})
	reg.MustRegister(openmp.LegacyInclude{})
	reg.MustRegister(openmp.NativeArgumentCount{})
	reg.MustRegister(openmp.DeprecatedNative{})
	reg.MustRegister(openmp.FormatArgumentCount{})
	reg.MustRegister(openmp.BufferSize{})
	reg.MustRegister(openmp.DiscardedResourceHandle{})
	reg.MustRegister(openmp.MismatchedResourceHandle{})
	reg.MustRegister(openmp.UnreleasedResourceHandle{})
	reg.MustRegister(openmp.OverwrittenResourceHandle{})
	reg.MustRegister(correctness.UnreachableCode{})
	reg.MustRegister(correctness.MissingReturnValue{})
	reg.MustRegister(suspicious.IdenticalBranches{})
	reg.MustRegister(suspicious.DeadWrite{})
	reg.MustRegister(correctness.PossiblyUninitialized{})
	reg.MustRegister(correctness.DuplicateFunctionDefinition{})
	reg.MustRegister(correctness.DuplicateGlobalDefinition{})
	reg.MustRegister(maintainability.UnusedFunction{})
	reg.MustRegister(maintainability.UnusedGlobal{})
	reg.MustRegister(suspicious.RecursiveCall{})
	reg.MustRegister(performance.LargeLocalArray{})
	reg.MustRegister(performance.RepeatedStrlen{})
	reg.MustRegister(suspicious.FloatEquality{})
	reg.MustRegister(openmp.NonPublicCallback{})
	reg.MustRegister(openmp.InvalidSentinelComparison{})
	reg.MustRegister(openmp.UnescapedSQLFormat{})
	reg.MustRegister(openmp.DiscardedRepeatingTimer{})
	reg.MustRegister(correctness.NonCallableSymbol{})
	reg.MustRegister(openmp.RawTickSubtraction{})
	reg.MustRegister(openmp.SscanfFormatArgumentCount{})
	reg.MustRegister(openmp.SetTimerExFormatArgumentCount{})
}

func Default() *lint.Registrar {
	reg := lint.NewRegistrar()
	Register(reg)
	return reg
}
