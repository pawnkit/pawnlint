package correctness

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type MissingReturnValue struct{}

func (MissingReturnValue) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "missing-return-value",
		Name:            "Missing return value",
		Summary:         "Reports value-returning functions with paths that return no value",
		Explanation:     "Once a function returns a value, every reachable exit should return a value. The rule reports bare returns and paths that reach the end of such a function.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"control-flow", "returns", "correctness"},
	}
}

func (MissingReturnValue) Run(ctx *lint.Context) {
	if ctx.Flow == nil {
		return
	}
	for _, function := range ctx.Flow.Functions {
		if function.Uncertain || !hasReachableValueReturn(ctx, function.Node) {
			continue
		}
		for _, statement := range ctx.Walk.OfKind(parser.KindReturnStatement) {
			if ctx.Walk.EnclosingFunction(statement) != function.Node || !function.Reachable(statement) || statement.Field("value") != nil {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  "return statement must return a value",
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(statement),
			})
		}
		if !function.CanFallThrough() {
			continue
		}
		name := function.Node.Field("name")
		ctx.Report(diagnostic.Diagnostic{
			Message:  "not all paths return a value",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(name),
		})
	}
}

func hasReachableValueReturn(ctx *lint.Context, function *parser.Node) bool {
	flow := ctx.Flow.Function(function)
	for _, statement := range ctx.Walk.OfKind(parser.KindReturnStatement) {
		if ctx.Walk.EnclosingFunction(statement) == function && statement.Field("value") != nil && flow.Reachable(statement) {
			return true
		}
	}
	return false
}
