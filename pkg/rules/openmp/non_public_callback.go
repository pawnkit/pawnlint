package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type NonPublicCallback struct{}

func (NonPublicCallback) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "non-public-callback",
		Name:            "Non-public callback",
		Summary:         "Reports functions named exactly like a callback but missing the public qualifier",
		Explanation:     explanationNonPublicCallback,
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"callbacks", "openmp", "samp", "api"},
	}
}

const explanationNonPublicCallback = `The server dispatches callbacks by looking up a ` + "`public`" + ` function with the
exact callback name; a same-named function without ` + "`public`" + ` compiles cleanly
but is never called:

` + "```pawn" + `
OnPlayerConnect(playerid)
{
    // never runs; the server calls the public symbol, which does not exist
}
` + "```" + `

The rule reports functions whose name is an exact, case-sensitive match for a
known callback but that lack the ` + "`public`" + ` qualifier. No fix is offered
because a same-named private helper may be intentional.`

func (NonPublicCallback) Run(ctx *lint.Context) {
	callbacks := ctx.Callbacks()
	ctx.Walk.IterKind(parser.KindFunctionDefinition, func(node *parser.Node) {
		if walk.HasChildToken(node, token.KwPublic) || ctx.Walk.Uncertain(node) {
			return
		}
		nameNode := node.Field("name")
		if nameNode == nil {
			return
		}
		name := ctx.Walk.Text(nameNode)
		if _, known := callbacks[name]; !known {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("function %q matches a known callback name but is not declared public, so it is never called", name),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(nameNode),
		})
	})
}
