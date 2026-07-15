package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type IgnoredReturnValue struct{}

func (IgnoredReturnValue) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "ignored-return-value",
		Name:            "Ignored return value",
		Summary:         "Reports discarded results from APIs marked must-use",
		Explanation:     "A native marked mustUse in API metadata requires callers to consume its result. Direct standalone calls are reported; nested, uncertain, macro-defined, and locally overridden calls are skipped.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"calls", "return-value", "api", "contracts"},
	}
}

func (IgnoredReturnValue) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindExpressionStatement, func(statement *parser.Node) {
		if statement.HasError || ctx.Walk.Uncertain(statement) {
			return
		}
		expression := unwrapParentheses(statement.Field("expression"))
		if expression == nil || expression.Kind != parser.KindCallExpression {
			return
		}
		native, name, ok := calledNative(ctx, expression)
		if !ok || !native.MustUse {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("return value from %q must be used", name),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(expression),
		})
	})
}
