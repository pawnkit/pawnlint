package maintainability

import (
	"strings"

	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type UnusedLocal struct{}

func (UnusedLocal) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unused-local",
		Name:            "Unused local",
		Summary:         "Reports local variables that are never referenced",
		Explanation:     explanationUnusedLocal,
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"unused", "variables", "semantic"},
	}
}

const explanationUnusedLocal = `A local variable that is never referenced adds noise and may indicate unfinished
code. Names beginning with ` + "`_`" + ` are treated as intentionally unused. The rule
does not offer a fix because an initializer may have side effects.`

func (UnusedLocal) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol.Kind != semantic.SymbolLocal || symbol.Ambiguous || strings.HasPrefix(symbol.Name, "_") {
			continue
		}
		if len(ctx.Semantic.References(symbol)) != 0 {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "local variable " + quoteName(symbol.Name) + " is never used",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(symbol.NameNode),
		})
	}
}

func quoteName(name string) string {
	return "\"" + strings.ReplaceAll(name, "\"", "\\\"") + "\""
}
