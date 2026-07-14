package suspicious

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type SuspiciousNegation struct{}

func (SuspiciousNegation) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "suspicious-negation",
		Name:            "Suspicious negation precedence",
		Summary:         "'!' binds tighter than &/|/^/==/!=, so !x & y is (!x) & y",
		Explanation:     explanationSuspiciousNegation,
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"precedence", "negation"},
	}
}

const explanationSuspiciousNegation = `'!' binds more tightly than bitwise and equality operators:

` + "```pawn" + `
!flags & FLAG_ADMIN
!value == expected
` + "```" + `

The likely forms are ` + "`!(flags & FLAG_ADMIN)`" + ` and ` + "`value != expected`" + `. No fix is
offered because the intended grouping cannot be known.`

func (SuspiciousNegation) Run(ctx *lint.Context) {
	m := ctx.Walk
	m.Iter(func(n *parser.Node) {
		if n.Kind != parser.KindBinaryExpression {
			return
		}
		if !misnegOp(n.Tok.Kind) {
			return
		}
		left := n.Field("left")
		if left == nil || left.Kind != parser.KindUnaryExpression {
			return
		}
		if left.Tok.Kind != token.Bang {
			return
		}
		if m.Uncertain(n) {
			return
		}
		d := diagnostic.Diagnostic{
			RuleID:   "suspicious-negation",
			Message:  "'!' binds tighter than '" + n.Tok.Kind.String() + "'; did you mean to parenthesise?",
			Filename: ctx.File.Path,
			Range:    m.Range(left),
			Notes: []diagnostic.RelatedLocation{{
				Range:   m.Range(n),
				Message: "consider '!(" + m.Text(n.Field("right")) + " …)' or a different operator",
			}},
		}
		ctx.Report(d)
	})
}

func misnegOp(k token.Kind) bool {
	switch k {
	case token.Amp, token.Pipe, token.Caret, token.Eq, token.NotEq,
		token.Shl, token.Shr, token.Ushr:
		return true
	default:
		return false
	}
}
