package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type ArgumentValueRange struct{}

func (ArgumentValueRange) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "argument-value-range",
		Name:            "Argument value range",
		Summary:         "Reports constant arguments outside API parameter bounds",
		Explanation:     "API metadata can define inclusive minimum and maximum values for scalar input parameters. The rule reports only definite constant violations and skips named, macro-structured, unresolved, and locally overridden calls.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"calls", "arguments", "api", "contracts", "range"},
	}
}

func (ArgumentValueRange) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindCallExpression, func(call *parser.Node) {
		native, name, ok := calledNative(ctx, call)
		if !ok {
			return
		}
		arguments := call.Field("arguments")
		if arguments == nil || arguments.HasError || hasNamedArgument(arguments) || len(arguments.Children) > len(native.Parameters) {
			return
		}
		for index, argument := range arguments.Children {
			parameter := native.Parameters[index]
			if parameter.Minimum == nil && parameter.Maximum == nil || argument.HasError || !argumentExpression(argument.Kind) {
				continue
			}
			value, known := ctx.Eval(argument)
			if !known || valueInRange(value, parameter) {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  argumentRangeMessage(name, index, parameter, value),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(argument),
			})
		}
	})
}

func valueInRange(value int64, parameter api.Parameter) bool {
	return (parameter.Minimum == nil || value >= *parameter.Minimum) && (parameter.Maximum == nil || value <= *parameter.Maximum)
}

func argumentRangeMessage(native string, index int, parameter api.Parameter, value int64) string {
	argument := fmt.Sprintf("argument %d", index+1)
	if parameter.Name != "" {
		argument = fmt.Sprintf("argument %q", parameter.Name)
	}
	switch {
	case parameter.Minimum != nil && parameter.Maximum != nil:
		return fmt.Sprintf("%s to %q must be between %d and %d, but is %d", argument, native, *parameter.Minimum, *parameter.Maximum, value)
	case parameter.Minimum != nil:
		return fmt.Sprintf("%s to %q must be at least %d, but is %d", argument, native, *parameter.Minimum, value)
	default:
		return fmt.Sprintf("%s to %q must be at most %d, but is %d", argument, native, *parameter.Maximum, value)
	}
}
