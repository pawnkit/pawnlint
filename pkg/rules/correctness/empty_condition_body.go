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
	m.Iter(func(n *parser.Node) {
		var body *parser.Node
		kw := ""
		switch n.Kind {
		case parser.KindIfStatement:
			body = n.Field("consequence")
			kw = "if"
		case parser.KindWhileStatement:
			body = n.Field("body")
			kw = "while"
		case parser.KindForStatement:
			body = n.Field("body")
			kw = "for"
		default:
			return
		}
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
	})
}
