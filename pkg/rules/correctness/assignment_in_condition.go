package correctness

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type AssignmentInCondition struct{}

func (AssignmentInCondition) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "assignment-in-condition",
		Name:            "Assignment in condition",
		Summary:         "An assignment used as an if/while condition is often a typo for ==",
		Explanation:     explanationAssignmentInCondition,
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"if", "while", "assignment"},
	}
}

const explanationAssignmentInCondition = `A direct assignment in an if or loop condition is often a typo:

` + "```pawn" + `
if (playerid = BAD_PLAYER)
` + "```" + `

Use ` + "`==`" + ` for comparison. If the assignment is intentional, wrap it in another
pair of parentheses. This rule has no fix because either meaning may be valid.`

func (AssignmentInCondition) Run(ctx *lint.Context) {
	m := ctx.Walk
	m.Iter(func(n *parser.Node) {
		var cond *parser.Node
		switch n.Kind {
		case parser.KindIfStatement, parser.KindWhileStatement:
			cond = n.Field("condition")
		case parser.KindDoWhileStatement:
			cond = n.Field("condition")
		default:
			return
		}
		if cond == nil {
			return
		}
		if m.Uncertain(n) {
			return
		}
		inner := unwrapParen(cond)
		if inner == nil || inner.Kind != parser.KindAssignmentExpression {
			return
		}
		if inner.Tok.Kind != token.Assign {
			return
		}
		if isDoubleParenOptOut(cond) {
			return
		}
		d := diagnostic.Diagnostic{
			RuleID:   "assignment-in-condition",
			Message:  "assignment used as a condition; did you mean '=='?",
			Filename: ctx.File.Path,
			Range:    m.Range(inner),
			Notes: []diagnostic.RelatedLocation{{
				Range:   m.Range(n),
				Message: "wrap the assignment in an extra pair of parentheses to silence this",
			}},
		}
		ctx.Report(d)
	})
}

func unwrapParen(n *parser.Node) *parser.Node {
	if n == nil {
		return nil
	}
	if n.Kind == parser.KindParenthesizedExpression {
		return n.Field("expression")
	}
	return n
}

func isDoubleParenOptOut(cond *parser.Node) bool {
	if cond == nil || cond.Kind != parser.KindParenthesizedExpression {
		return false
	}
	inner := cond.Field("expression")
	if inner == nil || inner.Kind != parser.KindParenthesizedExpression {
		return false
	}
	return true
}
