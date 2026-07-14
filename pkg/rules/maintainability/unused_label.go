package maintainability

import (
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type UnusedLabel struct{}

func (UnusedLabel) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unused-label",
		Name:            "Unused label",
		Summary:         "Reports labels that are not targeted by a goto statement",
		Explanation:     explanationUnusedLabel,
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         true,
		Tags:            []string{"unused", "labels", "semantic"},
	}
}

const explanationUnusedLabel = `A label with no matching ` + "`goto`" + ` target has no effect. The rule reports only
labels that resolve unambiguously within one function. The safe fix removes the
label and keeps the following statement unchanged.`

func (UnusedLabel) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol.Kind != semantic.SymbolLabel || symbol.Ambiguous || len(ctx.Semantic.References(symbol)) != 0 {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "label " + quoteName(symbol.Name) + " is never used",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(symbol.NameNode),
			Fix: &diagnostic.Fix{
				Description: "remove the unused label",
				Edits: []diagnostic.Edit{{
					Range:   ctx.Walk.Range(symbol.Decl),
					NewText: "",
				}},
			},
		})
	}
}
