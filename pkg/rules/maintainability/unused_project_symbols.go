package maintainability

import (
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type UnusedFunction struct{}

func (UnusedFunction) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unused-function",
		Name:            "Unused function",
		Summary:         "Reports internal functions unused by any translation unit",
		Explanation:     "An unreferenced internal function may be dead code. Main, public, stock, callback, command-handler, state-qualified, operator, and underscore-prefixed functions are skipped. Translation units containing parser error nodes are skipped because references may be hidden.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"unused", "functions", "project"},
	}
}

func (UnusedFunction) Run(ctx *lint.Context) {
	if ctx.Project == nil {
		return
	}
	file := ctx.Project.File(ctx.File.Path)
	if !ctx.Project.InProgram(file) || projectUnitHasErrors(ctx.Project, file) {
		return
	}
	callbacks := ctx.Callbacks()
	for _, symbol := range file.Semantic.Symbols {
		if symbol.Kind != semantic.SymbolFunction || symbol.Decl.Kind != parser.KindFunctionDefinition || symbol.Ambiguous {
			continue
		}
		if symbol.Name == "main" || strings.HasPrefix(symbol.Name, "_") || strings.HasPrefix(symbol.Name, "operator") {
			continue
		}
		storage := file.Walk.Text(symbol.Decl.Field("storage"))
		if storage != "" && storage != "static" {
			continue
		}
		if symbol.Decl.Field("state") != nil || walk.HasChildToken(symbol.Decl, token.KwStock) || hasExternalSignature(ctx, symbol.Decl) {
			continue
		}
		if _, callback := callbacks[symbol.Name]; callback {
			continue
		}
		declaration, ok := projectDeclaration(ctx.Project, file, symbol)
		if !ok || len(ctx.Project.References(declaration)) != 0 {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "function " + quoteName(symbol.Name) + " is never used",
			Filename: ctx.File.Path,
			Range:    file.Walk.Range(symbol.NameNode),
		})
	}
}

type UnusedGlobal struct{}

func (UnusedGlobal) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unused-global",
		Name:            "Unused global",
		Summary:         "Reports global variables unused by any translation unit",
		Explanation:     "An unreferenced global variable may be dead code. Public and underscore-prefixed globals are skipped. Initializers are not removed automatically because they may have side effects.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"unused", "variables", "project"},
	}
}

func (UnusedGlobal) Run(ctx *lint.Context) {
	if ctx.Project == nil {
		return
	}
	file := ctx.Project.File(ctx.File.Path)
	if !ctx.Project.InProgram(file) || projectUnitHasErrors(ctx.Project, file) {
		return
	}
	for _, symbol := range file.Semantic.Symbols {
		if symbol.Kind != semantic.SymbolGlobal || symbol.Ambiguous || strings.HasPrefix(symbol.Name, "_") {
			continue
		}
		declarationNode := file.Walk.Parent(symbol.Decl)
		if walk.HasChildToken(declarationNode, token.KwPublic) || walk.HasChildToken(declarationNode, token.KwStock) {
			continue
		}
		declaration, ok := projectDeclaration(ctx.Project, file, symbol)
		if !ok || len(ctx.Project.References(declaration)) != 0 {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "global variable " + quoteName(symbol.Name) + " is never used",
			Filename: ctx.File.Path,
			Range:    file.Walk.Range(symbol.NameNode),
		})
	}
}

func projectUnitHasErrors(model *project.Model, file *project.File) bool {
	for _, unit := range model.Units {
		contains := false
		for _, candidate := range unit.Files {
			if candidate == file {
				contains = true
				break
			}
		}
		if !contains {
			continue
		}
		for _, candidate := range unit.Files {
			if candidate.Parsed == nil || candidate.Parsed.Root == nil || candidate.Parsed.Broken || candidate.Parsed.Root.HasError {
				return true
			}
		}
	}
	return false
}

func projectDeclaration(model *project.Model, file *project.File, symbol *semantic.Symbol) (project.Declaration, bool) {
	for _, declaration := range model.Declarations[symbol.Name] {
		if declaration.File == file && declaration.Symbol == symbol {
			return declaration, true
		}
	}
	return project.Declaration{}, false
}
