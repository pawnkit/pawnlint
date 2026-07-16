package correctness

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type UnreachableCode struct{}

func (UnreachableCode) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unreachable-code",
		Name:            "Unreachable code",
		Summary:         "Reports statements that cannot be executed",
		Explanation:     "Code after an unconditional return, jump, or non-terminating loop cannot execute. The rule skips functions with malformed or uncertain control flow.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"control-flow", "unreachable", "correctness"},
	}
}

func (UnreachableCode) Run(ctx *lint.Context) {
	if ctx.Flow == nil {
		return
	}
	for _, function := range ctx.Flow.Functions {
		if function == nil || function.Uncertain || function.Node == nil {
			continue
		}
		previousUnreachable := false
		for _, child := range function.Node.Children {
			unreachableVisit(ctx, function, child, false, previousUnreachable)
			if walk.IsStatement(child) {
				previousUnreachable = !function.Reachable(child)
			}
		}
	}
}

func unreachableVisit(ctx *lint.Context, function *controlflow.Function, node *parser.Node, ancestorUnreachable, previousUnreachable bool) {
	if node == nil || node.Kind == parser.KindFunctionDefinition {
		return
	}
	statement := walk.IsStatement(node)
	reachable := function.Reachable(node)
	if statement && node.Kind != parser.KindEmptyStatement && !ctx.Walk.Inactive(node) && !reachable && !ancestorUnreachable && !previousUnreachable {
		ctx.Report(diagnostic.Diagnostic{
			Message:  "unreachable code",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
		})
	}
	childAncestorUnreachable := ancestorUnreachable || statement && !reachable
	childPreviousUnreachable := false
	for _, child := range node.Children {
		unreachableVisit(ctx, function, child, childAncestorUnreachable, childPreviousUnreachable)
		if walk.IsStatement(child) {
			childPreviousUnreachable = !function.Reachable(child)
		}
	}
}
