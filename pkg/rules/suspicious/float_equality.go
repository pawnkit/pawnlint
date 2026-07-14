package suspicious

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type FloatEquality struct{}

func (FloatEquality) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "float-equality",
		Name:            "Float equality",
		Summary:         "Reports Float values compared with == or !=",
		Explanation:     explanationFloatEquality,
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"float", "comparison", "semantic"},
	}
}

const explanationFloatEquality = `Float values accumulate rounding error, so ` + "`==`" + ` and ` + "`!=`" + ` rarely mean
what they appear to:

` + "```pawn" + `
if (Float:distance == 0.0)
` + "```" + `

Use a tolerance comparison, or round both sides with ` + "`floatround`" + ` first if an
exact match is really intended:

` + "```pawn" + `
if (floatround(distance) == floatround(target))
` + "```" + `

The rule does not report when either side is a direct ` + "`floatround(...)`" + ` call,
since rounding first is the standard way to compare floats exactly. No fix is
offered because the correct tolerance or rounding style depends on context.`

func (FloatEquality) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.IterKind(parser.KindBinaryExpression, func(node *parser.Node) {
		if node.Tok.Kind != token.Eq && node.Tok.Kind != token.NotEq || ctx.Walk.Uncertain(node) {
			return
		}
		left := node.Field("left")
		right := node.Field("right")
		if isFloatRoundCall(ctx, left) || isFloatRoundCall(ctx, right) {
			return
		}
		if !isFloatOperand(ctx, left) && !isFloatOperand(ctx, right) {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "Float values compared with '" + node.Tok.Kind.String() + "'; rounding error makes this unreliable",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
		})
	})
}

func isFloatOperand(ctx *lint.Context, node *parser.Node) bool {
	tags := ctx.Semantic.ExpressionTags(node)
	return len(tags) == 1 && tags[0] == "Float"
}

func isFloatRoundCall(ctx *lint.Context, node *parser.Node) bool {
	if node == nil || node.Kind != parser.KindCallExpression {
		return false
	}
	callee := node.Field("function")
	return callee != nil && callee.Kind == parser.KindIdentifier && ctx.Walk.Text(callee) == "floatround"
}
