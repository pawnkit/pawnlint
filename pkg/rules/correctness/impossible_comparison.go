package correctness

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type ImpossibleComparison struct{}

func (ImpossibleComparison) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "impossible-comparison",
		Name:            "Impossible comparison",
		Summary:         "Reports comparisons that cannot produce both results",
		Explanation:     "Definite ranges from boolean expressions, remainders, bit masks, unsigned shifts, and conditional expressions prove when a comparison is always true or false. Unknown, floating-point, overflowing, macro-derived, and malformed expressions are ignored.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"comparisons", "ranges", "conditions", "semantic"},
	}
}

type comparisonRange struct {
	minimum int64
	maximum int64
}

func (r comparisonRange) exact() bool {
	return r.minimum == r.maximum
}

func (ImpossibleComparison) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, node := range ctx.Walk.OfKind(parser.KindBinaryExpression) {
		if !comparisonOperator(node.Tok.Kind) || impossibleComparisonSkipped(ctx, node) || chainedRelationalComparison(ctx, node) {
			continue
		}
		left, leftOK := expressionComparisonRange(ctx, node.Field("left"))
		right, rightOK := expressionComparisonRange(ctx, node.Field("right"))
		if !leftOK || !rightOK || left.exact() && right.exact() {
			continue
		}
		result, known := comparisonResult(node.Tok.Kind, left, right)
		if !known {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("comparison %q is always %t", node.Tok.Kind.String(), result),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
		})
	}
}

func chainedRelationalComparison(ctx *lint.Context, node *parser.Node) bool {
	if !relationalOperator(node.Tok.Kind) {
		return false
	}
	parent := ctx.Walk.Parent(node)
	if parent != nil && parent.Kind == parser.KindBinaryExpression && relationalOperator(parent.Tok.Kind) {
		return true
	}
	for _, child := range []*parser.Node{node.Field("left"), node.Field("right")} {
		if child != nil && child.Kind == parser.KindBinaryExpression && relationalOperator(child.Tok.Kind) {
			return true
		}
	}
	return false
}

func relationalOperator(kind token.Kind) bool {
	switch kind {
	case token.Lt, token.Gt, token.LtEq, token.GtEq:
		return true
	default:
		return false
	}
}

func comparisonOperator(kind token.Kind) bool {
	switch kind {
	case token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq:
		return true
	default:
		return false
	}
}

func impossibleComparisonSkipped(ctx *lint.Context, node *parser.Node) bool {
	if node == nil || node.HasError || node.Tok.Origin != nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return true
	}
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindDirectiveDefine, parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock,
			parser.KindSharedConditional, parser.KindConditionalFunction, parser.KindConditionalSplice:
			return true
		}
	}
	return false
}

func expressionComparisonRange(ctx *lint.Context, node *parser.Node) (comparisonRange, bool) {
	if node == nil || node.HasError || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return comparisonRange{}, false
	}
	if value, ok := ctx.Constant(node); ok {
		return comparisonRange{minimum: value, maximum: value}, true
	}
	switch node.Kind {
	case parser.KindParenthesizedExpression, parser.KindTaggedExpression:
		return expressionComparisonRange(ctx, node.Field("expression"))
	case parser.KindUnaryExpression:
		if node.Tok.Kind == token.Bang {
			return comparisonRange{minimum: 0, maximum: 1}, true
		}
		inner, ok := expressionComparisonRange(ctx, node.Field("expression"))
		if !ok {
			return comparisonRange{}, false
		}
		switch node.Tok.Kind {
		case token.Plus:
			return inner, true
		case token.Minus:
			if inner.minimum <= -2147483648 {
				return comparisonRange{}, false
			}
			return comparisonRange{minimum: -inner.maximum, maximum: -inner.minimum}, true
		}
	case parser.KindBinaryExpression:
		if comparisonOperator(node.Tok.Kind) || node.Tok.Kind == token.AndAnd || node.Tok.Kind == token.OrOr {
			return comparisonRange{minimum: 0, maximum: 1}, true
		}
		return binaryComparisonRange(ctx, node)
	case parser.KindTernaryExpression:
		consequence, consequenceOK := expressionComparisonRange(ctx, node.Field("consequence"))
		alternative, alternativeOK := expressionComparisonRange(ctx, node.Field("alternative"))
		if !consequenceOK || !alternativeOK {
			return comparisonRange{}, false
		}
		return comparisonRange{minimum: min(consequence.minimum, alternative.minimum), maximum: max(consequence.maximum, alternative.maximum)}, true
	case parser.KindExpressionList:
		if len(node.Children) != 0 {
			return expressionComparisonRange(ctx, node.Children[len(node.Children)-1])
		}
	}
	return comparisonRange{}, false
}

func binaryComparisonRange(ctx *lint.Context, node *parser.Node) (comparisonRange, bool) {
	leftNode := node.Field("left")
	rightNode := node.Field("right")
	switch node.Tok.Kind {
	case token.Percent:
		divisor, ok := ctx.Constant(rightNode)
		if !ok || divisor == 0 || divisor == -2147483648 {
			return comparisonRange{}, false
		}
		if divisor < 0 {
			divisor = -divisor
		}
		result := comparisonRange{minimum: -divisor + 1, maximum: divisor - 1}
		if left, known := expressionComparisonRange(ctx, leftNode); known {
			if left.minimum >= 0 {
				result.minimum = 0
			}
			if left.maximum <= 0 {
				result.maximum = 0
			}
		}
		return result, true
	case token.Amp:
		if mask, ok := nonnegativeComparisonConstant(ctx, rightNode); ok {
			return comparisonRange{minimum: 0, maximum: mask}, true
		}
		if mask, ok := nonnegativeComparisonConstant(ctx, leftNode); ok {
			return comparisonRange{minimum: 0, maximum: mask}, true
		}
	case token.Ushr:
		shift, ok := ctx.Constant(rightNode)
		if !ok || shift < 1 || shift > 31 {
			return comparisonRange{}, false
		}
		return comparisonRange{minimum: 0, maximum: int64(uint64(1)<<(32-uint(shift))) - 1}, true
	}
	return comparisonRange{}, false
}

func nonnegativeComparisonConstant(ctx *lint.Context, node *parser.Node) (int64, bool) {
	value, ok := ctx.Constant(node)
	return value, ok && value >= 0
}

func comparisonResult(kind token.Kind, left, right comparisonRange) (bool, bool) {
	switch kind {
	case token.Eq:
		if left.maximum < right.minimum || right.maximum < left.minimum {
			return false, true
		}
	case token.NotEq:
		if left.maximum < right.minimum || right.maximum < left.minimum {
			return true, true
		}
	case token.Lt:
		if left.maximum < right.minimum {
			return true, true
		}
		if left.minimum >= right.maximum {
			return false, true
		}
	case token.Gt:
		if left.minimum > right.maximum {
			return true, true
		}
		if left.maximum <= right.minimum {
			return false, true
		}
	case token.LtEq:
		if left.maximum <= right.minimum {
			return true, true
		}
		if left.minimum > right.maximum {
			return false, true
		}
	case token.GtEq:
		if left.minimum >= right.maximum {
			return true, true
		}
		if left.maximum < right.minimum {
			return false, true
		}
	}
	return false, false
}
