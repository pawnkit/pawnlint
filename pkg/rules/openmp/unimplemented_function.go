package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type UnimplementedFunction struct{}

func (UnimplementedFunction) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unimplemented-function",
		Name:            "Unimplemented function",
		Summary:         "Reports legacy API calls intentionally not implemented by open.mp",
		Explanation:     "The official open.mp includes retain removed SA-MP functions as forward declarations so calls fail with a specific compiler error. The rule reports those direct calls and includes replacement guidance when available.",
		Category:        diagnostic.CategoryOpenMP,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"migration", "compatibility", "api"},
	}
}

func (UnimplementedFunction) Run(ctx *lint.Context) {
	if ctx.Target == "samp" {
		return
	}
	unsupported := api.UnsupportedFunctions("openmp")
	ctx.Walk.IterKind(parser.KindCallExpression, func(call *parser.Node) {
		if call.HasError || ctx.Walk.Uncertain(call) {
			return
		}
		callee := call.Field("function")
		if callee == nil || callee.Kind != parser.KindIdentifier || callee.HasError {
			return
		}
		name := ctx.Walk.Text(callee)
		entry, ok := unsupported[name]
		if !ok || ctx.Semantic.Resolve(callee) != nil || projectDefinesName(ctx, name) || projectImplementsFunction(ctx, name) {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:     fmt.Sprintf("function %q is not implemented by open.mp", name),
			Filename:    ctx.File.Path,
			Range:       ctx.Walk.Range(callee),
			Suggestions: suggestions(entry.Suggested),
		})
	})
}

func projectImplementsFunction(ctx *lint.Context, name string) bool {
	if ctx.Project == nil {
		return false
	}
	for _, declaration := range ctx.Project.Declarations[name] {
		if declaration.Kind == semantic.SymbolFunction && !walk.HasChildToken(declaration.Node, token.KwForward) {
			return true
		}
	}
	return false
}
