package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DiscardedRepeatingTimer struct{}

func (DiscardedRepeatingTimer) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "discarded-repeating-timer",
		Name:            "Discarded repeating timer",
		Summary:         "Reports repeating SetTimer/SetTimerEx handles discarded before they can be killed",
		Explanation:     explanationDiscardedRepeatingTimer,
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"timer", "resource", "leak"},
	}
}

const explanationDiscardedRepeatingTimer = `A repeating timer runs forever unless its handle is passed to KillTimer.
Discarding the handle immediately, as a standalone statement, means nothing
can ever stop it:

` + "```pawn" + `
SetTimer("Tick", 1000, true);
` + "```" + `

Store the handle so it can be killed later:

` + "```pawn" + `
tickTimer = SetTimer("Tick", 1000, true);
` + "```" + `

The rule reports only direct standalone calls whose ` + "`repeating`" + ` argument is
the constant ` + "`true`" + `. A one-shot timer (` + "`repeating`" + ` is ` + "`false`" + `) needs no handle,
and is not reported. No fix is offered because a name for the stored handle
cannot be inferred.`

func (DiscardedRepeatingTimer) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.IterKind(parser.KindExpressionStatement, func(statement *parser.Node) {
		if statement.HasError || ctx.Walk.Uncertain(statement) {
			return
		}
		call := unwrapParentheses(statement.Field("expression"))
		if call == nil || call.Kind != parser.KindCallExpression {
			return
		}
		_, name, ok := calledNative(ctx, call)
		if !ok || name != "SetTimer" && name != "SetTimerEx" {
			return
		}
		arguments := call.Field("arguments")
		if arguments == nil || len(arguments.Children) < 3 {
			return
		}
		repeating, known := ctx.Eval(arguments.Children[2])
		if !known || repeating == 0 {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("repeating timer created by %q is discarded; store its handle for KillTimer", name),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(call),
		})
	})
}
