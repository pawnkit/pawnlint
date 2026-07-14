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
	for _, node := range ctx.Walk.All() {
		if !walk.IsStatement(node) || node.Kind == parser.KindEmptyStatement || ctx.Walk.Inactive(node) {
			continue
		}
		functionNode := ctx.Walk.EnclosingFunction(node)
		function := ctx.Flow.Function(functionNode)
		if function == nil || function.Uncertain || function.Reachable(node) {
			continue
		}
		if unreachableAncestor(ctx, function, node) || unreachablePrevious(ctx, function, node) {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "unreachable code",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
		})
	}
}

func unreachableAncestor(ctx *lint.Context, function *controlflow.Function, node *parser.Node) bool {
	for _, ancestor := range ctx.Walk.Ancestors(node) {
		if ancestor == function.Node {
			return false
		}
		if walk.IsStatement(ancestor) && !function.Reachable(ancestor) {
			return true
		}
	}
	return false
}

func unreachablePrevious(ctx *lint.Context, function *controlflow.Function, node *parser.Node) bool {
	parent := ctx.Walk.Parent(node)
	if parent == nil {
		return false
	}
	for i, sibling := range parent.Children {
		if sibling != node {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			previous := parent.Children[j]
			if !walk.IsStatement(previous) {
				continue
			}
			return !function.Reachable(previous)
		}
		return false
	}
	return false
}
