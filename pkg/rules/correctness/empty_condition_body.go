package correctness

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type EmptyConditionBody struct{}

func (EmptyConditionBody) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "empty-condition-body",
		Name:            "Empty body after condition",
		Summary:         "Accidental semicolon after an if/while/for condition makes the following block unconditional",
		Explanation:     explanationEmptyConditionBody,
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  true,
		Fixable:         true,
		Tags:            []string{"if", "while", "for"},
	}
}

const explanationEmptyConditionBody = `A semicolon after a condition creates an empty body. The following block then
runs unconditionally:

` + "```pawn" + `
if (connected);
{
    Kick(playerid);
}
` + "```" + `

The rule reports this only when a block follows the empty body. The safe fix
removes the semicolon.`

func (EmptyConditionBody) Run(ctx *lint.Context) {
	m := ctx.Walk
	check := func(n *parser.Node, body *parser.Node, kw string) {
		if body == nil || body.Kind != parser.KindEmptyStatement {
			return
		}
		if body.Start == body.End {
			return
		}
		if m.Uncertain(n) {
			return
		}
		sib := m.NextSibling(n)
		if sib == nil || sib.Kind != parser.KindBlock {
			return
		}
		d := diagnostic.Diagnostic{
			RuleID:   "empty-condition-body",
			Message:  "stray ';' after " + kw + " condition makes the following block unconditional",
			Filename: ctx.File.Path,
			Range:    m.Range(body),
			Notes: []diagnostic.RelatedLocation{{
				Range:   m.Range(sib),
				Message: "this block runs unconditionally",
			}},
		}
		d.Fix = &diagnostic.Fix{
			Description: "remove the stray semicolon",
			Edits: []diagnostic.Edit{{
				Range:   m.Range(body),
				NewText: "",
			}},
		}
		ctx.Report(d)
	}
	m.IterKind(parser.KindIfStatement, func(n *parser.Node) { check(n, n.Field("consequence"), "if") })
	m.IterKind(parser.KindWhileStatement, func(n *parser.Node) { check(n, n.Field("body"), "while") })
	m.IterKind(parser.KindForStatement, func(n *parser.Node) { check(n, n.Field("body"), "for") })
}
