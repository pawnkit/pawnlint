package openmp

import (
	"fmt"
	"path/filepath"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DeprecatedFunction struct{}

func (DeprecatedFunction) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "deprecated-function",
		Name:            "Deprecated function",
		Summary:         "Reports deprecated compatibility functions in open.mp",
		Explanation:     "Some legacy SA-MP APIs remain available as compatibility stocks but are documented as broken by the official open.mp includes. The rule reports direct calls and includes the official guidance.",
		Category:        diagnostic.CategoryOpenMP,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"deprecated", "migration", "compatibility", "api"},
	}
}

func (DeprecatedFunction) Run(ctx *lint.Context) {
	if ctx.Target == "samp" {
		return
	}
	deprecated := api.DeprecatedFunctions("openmp")
	ctx.Walk.IterKind(parser.KindCallExpression, func(call *parser.Node) {
		if call.HasError || ctx.Walk.Uncertain(call) {
			return
		}
		callee := call.Field("function")
		if callee == nil || callee.Kind != parser.KindIdentifier || callee.HasError {
			return
		}
		name := ctx.Walk.Text(callee)
		entry, ok := deprecated[name]
		if !ok || ctx.Semantic.Resolve(callee) != nil || projectDefinesName(ctx, name) || projectOverridesDeprecatedFunction(ctx, name) {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:   fmt.Sprintf("function %q is deprecated", name),
			Filename:  ctx.File.Path,
			Range:     ctx.Walk.Range(callee),
			Suggested: entry.Suggested,
		})
	})
}

func projectOverridesDeprecatedFunction(ctx *lint.Context, name string) bool {
	if ctx.Project == nil {
		return false
	}
	for _, declaration := range ctx.Project.Declarations[name] {
		if declaration.Kind == semantic.SymbolFunction && filepath.Base(declaration.File.Path) != "_open_mp.inc" {
			return true
		}
	}
	return false
}
