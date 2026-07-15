package correctness

import (
	"fmt"
	"path/filepath"

	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type ConflictingIncludeSymbol struct{}

func (ConflictingIncludeSymbol) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "conflicting-include-symbol",
		Name:            "Conflicting include symbol",
		Summary:         "Reports namespace collisions contributed by included files",
		Explanation:     "Functions, globals, enum names, and enum entries share Pawn namespaces in combinations that can collide across files. Duplicate function and global definitions remain owned by their dedicated rules.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"symbols", "project", "includes", "namespaces"},
	}
}

func (ConflictingIncludeSymbol) Run(ctx *lint.Context) {
	if ctx.Project == nil {
		return
	}
	current := ctx.Project.File(ctx.File.Path)
	for _, conflict := range ctx.Project.ConflictingIncludeSymbols() {
		if current == nil || conflict.Owner != current {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("%s %q conflicts with %s from %q", symbolKind(conflict.Second), conflict.Name, symbolKind(conflict.First), filepath.Base(conflict.First.File.Path)),
			Filename: conflict.Second.File.Path,
			Range:    conflict.Second.File.Walk.Range(conflict.Second.Symbol.NameNode),
		})
	}
}

func symbolKind(declaration project.Declaration) string {
	switch declaration.Kind {
	case semantic.SymbolFunction:
		return "function"
	case semantic.SymbolGlobal:
		if declaration.Symbol.Constant {
			return "global constant"
		}
		return "global variable"
	case semantic.SymbolEnumRoot:
		return "enum name"
	case semantic.SymbolEnumEntry:
		return "enum entry"
	default:
		return "symbol"
	}
}
