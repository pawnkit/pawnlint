package suspicious

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type SuspiciousCommaExpression struct{}

func (SuspiciousCommaExpression) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "suspicious-comma-expression",
		Name:            "Suspicious comma expression",
		Summary:         "The comma operator chains sub-expressions; it is rarely intended in statements or returns",
		Explanation:     explanationSuspiciousCommaExpression,
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"comma", "readability"},
	}
}

const explanationSuspiciousCommaExpression = `The comma operator evaluates each expression but keeps only the last value:

` + "```pawn" + `
return First(), Second();
` + "```" + `

The rule checks comma expressions used as statements or return values. It does
not report argument lists, declarations, initializers, or for clauses. No fix
is offered because the intended result is unknown.`

func (SuspiciousCommaExpression) Run(ctx *lint.Context) {
	m := ctx.Walk
	m.IterKind(parser.KindExpressionList, func(n *parser.Node) {
		if countExpressionList(n) < 2 {
			return
		}
		parent := m.Parent(n)
		if parent == nil {
			return
		}
		switch parent.Kind {
		case parser.KindExpressionStatement:
			if assignmentSequence(n) {
				return
			}
		case parser.KindReturnStatement:
		default:
			return
		}
		if m.Uncertain(n) {
			return
		}
		d := diagnostic.Diagnostic{
			RuleID:   "suspicious-comma-expression",
			Message:  "comma operator chains sub-expressions; consider separate statements",
			Filename: ctx.File.Path,
			Range:    m.Range(n),
		}
		ctx.Report(d)
	})
}

func assignmentSequence(n *parser.Node) bool {
	if n == nil {
		return false
	}
	if n.Kind == parser.KindAssignmentExpression || n.Kind == parser.KindUpdateExpression {
		return true
	}
	if n.Kind != parser.KindExpressionList || len(n.Children) == 0 {
		return false
	}
	for _, child := range n.Children {
		if !assignmentSequence(child) {
			return false
		}
	}
	return true
}

func countExpressionList(n *parser.Node) int {
	if n == nil || n.Kind != parser.KindExpressionList {
		return 0
	}
	return len(n.Children)
}
