package suspicious

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type ComparisonChain struct{}

func (ComparisonChain) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "comparison-chain",
		Name:            "Chained comparison",
		Summary:         "Chained relational comparisons (a < b < c) do not test a range",
		Explanation:     explanationComparisonChain,
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"comparison", "range"},
	}
}

const explanationComparisonChain = `Pawn evaluates chained comparisons from left to right:

` + "```pawn" + `
0 < value < 10
` + "```" + `

This compares the first result, 0 or 1, with 10. Write the range test as:

` + "```pawn" + `
0 < value && value < 10
` + "```" + `

No fix is offered because complex expressions may need manual changes.`

func (ComparisonChain) Run(ctx *lint.Context) {
	m := ctx.Walk
	m.IterKind(parser.KindBinaryExpression, func(n *parser.Node) {
		if !isRelational(n.Tok.Kind) {
			return
		}
		left := n.Field("left")
		right := n.Field("right")
		chained := false
		if left != nil && left.Kind == parser.KindBinaryExpression && isRelational(left.Tok.Kind) {
			chained = true
		}
		if right != nil && right.Kind == parser.KindBinaryExpression && isRelational(right.Tok.Kind) {
			chained = true
		}
		if !chained {
			return
		}
		if m.Uncertain(n) {
			return
		}
		d := diagnostic.Diagnostic{
			RuleID:   "comparison-chain",
			Message:  "chained comparison does not test a range; join with &&",
			Filename: ctx.File.Path,
			Range:    m.Range(n),
			Notes: []diagnostic.RelatedLocation{{
				Range:   m.Range(n),
				Message: "use 'a < b && b < c' instead",
			}},
		}
		ctx.Report(d)
	})
}

func isRelational(k token.Kind) bool {
	switch k {
	case token.Lt, token.Gt, token.LtEq, token.GtEq:
		return true
	default:
		return false
	}
}
