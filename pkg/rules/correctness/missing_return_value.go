package correctness

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/controlflow"
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
	returns := make(map[*parser.Node][]*parser.Node)
	for _, statement := range ctx.Walk.OfKind(parser.KindReturnStatement) {
		function := ctx.Walk.EnclosingFunction(statement)
		returns[function] = append(returns[function], statement)
	}
	for _, function := range ctx.Flow.Functions {
		statements := returns[function.Node]
		if function.Uncertain || !hasReachableValueReturn(function, statements) {
			continue
		}
		for _, statement := range statements {
			if !function.Reachable(statement) || statement.Field("value") != nil {
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

func hasReachableValueReturn(function *controlflow.Function, statements []*parser.Node) bool {
	for _, statement := range statements {
		if statement.Field("value") != nil && function.Reachable(statement) {
			return true
		}
	}
	return false
}
