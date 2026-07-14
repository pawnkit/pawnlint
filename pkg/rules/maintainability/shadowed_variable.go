package maintainability

import (
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type ShadowedVariable struct{}

func (ShadowedVariable) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "shadowed-variable",
		Name:            "Shadowed variable",
		Summary:         "Reports local declarations that hide an outer variable",
		Explanation:     explanationShadowedVariable,
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"variables", "shadowing", "semantic"},
	}
}

const explanationShadowedVariable = `A local variable with the same name as an outer variable can make code hard to
follow. The rule reports only unambiguous bindings and does not offer a rename
because related references may span more code.`

func (ShadowedVariable) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol.Kind != semantic.SymbolLocal && symbol.Kind != semantic.SymbolParameter {
			continue
		}
		outer := ctx.Semantic.Shadowed(symbol)
		if outer == nil || outer.Kind == semantic.SymbolFunction || outer.Kind == semantic.SymbolEnumEntry {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  quoteName(symbol.Name) + " shadows an outer variable",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(symbol.NameNode),
			Notes: []diagnostic.RelatedLocation{{
				Range:   ctx.Walk.Range(outer.NameNode),
				Message: "outer declaration is here",
			}},
		})
	}
}
