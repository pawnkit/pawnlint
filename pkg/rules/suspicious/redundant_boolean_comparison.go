package suspicious

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type RedundantBooleanComparison struct{}

func (RedundantBooleanComparison) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "redundant-boolean-comparison",
		Name:            "Redundant boolean comparison",
		Summary:         "Reports boolean expressions compared with true or false",
		Explanation:     "A boolean expression does not need to be compared with a boolean literal. Use the expression directly or negate it.",
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"boolean", "comparison", "semantic"},
	}
}

func (RedundantBooleanComparison) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.IterKind(parser.KindBinaryExpression, func(node *parser.Node) {
		if node.Tok.Kind != token.Eq && node.Tok.Kind != token.NotEq || ctx.Walk.Uncertain(node) {
			return
		}
		left := node.Field("left")
		right := node.Field("right")
		literal, other := left, right
		value, ok := ctx.Semantic.BooleanLiteral(literal)
		if !ok {
			literal, other = right, left
			value, ok = ctx.Semantic.BooleanLiteral(literal)
		}
		if !ok || !ctx.Semantic.Boolean(other) {
			return
		}
		negated := value == (node.Tok.Kind == token.NotEq)
		message := "boolean comparison is redundant; use the expression directly"
		if negated {
			message = "boolean comparison is redundant; negate the expression"
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  message,
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
		})
	})
}
