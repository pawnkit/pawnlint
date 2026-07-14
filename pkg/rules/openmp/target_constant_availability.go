package openmp

import (
	"fmt"

	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type TargetConstantAvailability struct{}

func (TargetConstantAvailability) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "target-constant-availability",
		Name:            "Target constant availability",
		Summary:         "Reports open.mp-only constants when targeting SA-MP",
		Explanation:     "The selected target determines which official constants are available. The rule reports unresolved value references declared only by open.mp modules and skips names declared by the project.",
		Category:        diagnostic.CategoryOpenMP,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"constant", "target", "migration", "api"},
	}
}

func (TargetConstantAvailability) Run(ctx *lint.Context) {
	if ctx.Target != "samp" {
		return
	}
	samp := ctx.Constants()
	openmp := api.Constants("openmp")
	for _, reference := range ctx.Semantic.UnresolvedReferences() {
		if reference.Target != semantic.ReferenceValue || ctx.Walk.Uncertain(reference.Node) {
			continue
		}
		name := ctx.Walk.Text(reference.Node)
		if _, ok := samp[name]; ok {
			continue
		}
		constant, ok := openmp[name]
		if !ok || !constant.OpenMPOnly || projectDeclaresName(ctx, name) || projectDefinesName(ctx, name) {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("constant %q is only available for the open.mp target", name),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(reference.Node),
		})
	}
}
