package maintainability

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type BooleanComplexity struct{}

func (BooleanComplexity) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "boolean-complexity",
		Name:            "Boolean complexity",
		Summary:         "Reports boolean expressions with too many logical operators",
		Explanation:     "Each maximal expression chain counts its && and || operators. Parentheses, boolean negation, and tag wrappers remain part of the same chain, while nested ternary branches and comparisons are checked independently. Inactive and uncertain syntax is ignored.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"complexity", "boolean", "maintainability"},
		Options: []lint.Option{{
			Name: "maximum", Summary: "Maximum logical operators per expression",
			Type: lint.OptionInteger, Default: int64(3), Minimum: 1, Maximum: 1000, HasMinimum: true, HasMaximum: true,
		}},
	}
}

func (BooleanComplexity) Run(ctx *lint.Context) {
	maximum := configuredBooleanComplexityMaximum(ctx)
	for _, node := range ctx.Walk.OfKind(parser.KindBinaryExpression) {
		if !booleanComplexityOperator(node) || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) || !booleanComplexityRoot(ctx, node) {
			continue
		}
		operators := booleanComplexityCount(ctx, node)
		if operators <= maximum {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("boolean expression contains %d logical operators, exceeding the maximum of %d", operators, maximum),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
		})
	}
}

func configuredBooleanComplexityMaximum(ctx *lint.Context) int {
	if ctx.PerRule != nil && ctx.PerRule["boolean-complexity"] != nil {
		if value, ok := ctx.PerRule["boolean-complexity"]["maximum"].(int64); ok && value > 0 {
			return int(value)
		}
	}
	return 3
}

func booleanComplexityOperator(node *parser.Node) bool {
	return node != nil && node.Kind == parser.KindBinaryExpression && (node.Tok.Kind == token.AndAnd || node.Tok.Kind == token.OrOr)
}

func booleanComplexityRoot(ctx *lint.Context, node *parser.Node) bool {
	current := node
	for parent := ctx.Walk.Parent(current); parent != nil; parent = ctx.Walk.Parent(current) {
		if booleanComplexityOperator(parent) {
			return false
		}
		if !booleanComplexityWrapper(parent, current) {
			return true
		}
		current = parent
	}
	return true
}

func booleanComplexityWrapper(parent, child *parser.Node) bool {
	if parent == nil || child == nil || parent.Field("expression") != child {
		return false
	}
	switch parent.Kind {
	case parser.KindParenthesizedExpression, parser.KindTaggedExpression:
		return true
	case parser.KindUnaryExpression:
		return parent.Tok.Kind == token.Bang
	default:
		return false
	}
}

func booleanComplexityCount(ctx *lint.Context, node *parser.Node) int {
	if node == nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return 0
	}
	if booleanComplexityOperator(node) {
		return 1 + booleanComplexityCount(ctx, node.Field("left")) + booleanComplexityCount(ctx, node.Field("right"))
	}
	if expression := node.Field("expression"); booleanComplexityWrapper(node, expression) {
		return booleanComplexityCount(ctx, expression)
	}
	return 0
}
