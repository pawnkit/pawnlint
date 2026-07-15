package openmp

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type RawTickSubtraction struct{}

func (RawTickSubtraction) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "raw-tick-subtraction",
		Name:            "Raw tick-count subtraction",
		Summary:         "Reports GetTickCount() subtracted directly instead of through a wraparound-safe helper",
		Explanation:     explanationRawTickSubtraction,
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"timing", "overflow", "samp", "openmp"},
	}
}

const explanationRawTickSubtraction = `GetTickCount() returns a 32-bit millisecond counter that wraps around after
about 24.8 days of server uptime. Subtracting two tick values directly breaks
on that wraparound:

` + "```pawn" + `
if (GetTickCount() - lastAction > 1000) // wrong after ~24.8 days uptime
` + "```" + `

Use a wraparound-safe difference helper (such as sscanf's bundled
GetTickCountDifference, or open.mp's Time_GetTickDifference) instead.`

func (RawTickSubtraction) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindBinaryExpression, func(node *parser.Node) {
		if node.Tok.Kind != token.Minus {
			return
		}
		if ctx.Walk.Uncertain(node) {
			return
		}
		left := isGetTickCountCall(ctx, node.Field("left"))
		right := isGetTickCountCall(ctx, node.Field("right"))
		if !left && !right {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "GetTickCount() subtracted directly; use a wraparound-safe tick difference helper instead",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
		})
	})
}

func isGetTickCountCall(ctx *lint.Context, node *parser.Node) bool {
	node = unwrapParentheses(node)
	if node == nil || node.Kind != parser.KindCallExpression {
		return false
	}
	callee := node.Field("function")
	return callee != nil && callee.Kind == parser.KindIdentifier && ctx.Walk.Text(callee) == "GetTickCount"
}
