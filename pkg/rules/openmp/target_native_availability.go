package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type TargetNativeAvailability struct{}

func (TargetNativeAvailability) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "target-native-availability",
		Name:            "Target native availability",
		Summary:         "Reports open.mp-only native calls when targeting SA-MP",
		Explanation:     "The selected target determines which official natives are available. The rule reports direct calls to natives declared only by open.mp modules and skips names declared by the project.",
		Category:        diagnostic.CategoryOpenMP,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"native", "target", "migration", "api"},
	}
}

func (TargetNativeAvailability) Run(ctx *lint.Context) {
	if ctx.Target != "samp" {
		return
	}
	samp := ctx.Natives()
	openmp := api.Natives("openmp")
	ctx.Walk.IterKind(parser.KindCallExpression, func(node *parser.Node) {
		if node == nil || node.HasError || ctx.Walk.Uncertain(node) {
			return
		}
		callee := node.Field("function")
		if callee == nil || callee.Kind != parser.KindIdentifier || callee.HasError {
			return
		}
		name := ctx.Walk.Text(callee)
		if _, ok := samp[name]; ok {
			return
		}
		native, ok := openmp[name]
		if !ok || !native.OpenMPOnly || projectDeclaresNativeName(ctx, callee, name) {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("native %q is only available for the open.mp target", name),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(callee),
		})
	})
}

func projectDeclaresNativeName(ctx *lint.Context, callee *parser.Node, name string) bool {
	if symbol := ctx.Semantic.Resolve(callee); symbol != nil {
		return true
	}
	if ctx.Project == nil {
		return false
	}
	return len(ctx.Project.Declarations[name]) != 0
}
