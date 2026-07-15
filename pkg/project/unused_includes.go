package project

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func (m *Model) UnusedIncludes() []IncludeIssue {
	if m == nil {
		return nil
	}
	return append([]IncludeIssue(nil), m.unusedIncludes...)
}

func (m *Model) buildUnusedIncludes() []IncludeIssue {
	var result []IncludeIssue
	for _, issue := range m.includeDirectives {
		include := issue.Include
		if include == nil || include.Resolved == nil || include.Optional || m.reachableWithout(issue) {
			continue
		}
		files := includeClosure(include.Resolved)
		if !safeUnusedIncludeFiles(files) || m.includeClosureReferenced(files) {
			continue
		}
		result = append(result, issue)
	}
	sortIncludeIssues(result)
	return result
}

func (m *Model) reachableWithout(issue IncludeIssue) bool {
	visited := make(map[*File]bool)
	var visit func(*File) bool
	visit = func(file *File) bool {
		if file == nil || visited[file] {
			return false
		}
		visited[file] = true
		if file == issue.Include.Resolved {
			return true
		}
		for _, include := range file.Includes {
			if file == issue.File && include == issue.Include {
				continue
			}
			if include != nil && !include.Uncertain && visit(include.Resolved) {
				return true
			}
		}
		return false
	}
	return visit(issue.Owner)
}

func includeClosure(root *File) map[*File]struct{} {
	files := make(map[*File]struct{})
	var visit func(*File)
	visit = func(file *File) {
		if file == nil {
			return
		}
		if _, exists := files[file]; exists {
			return
		}
		files[file] = struct{}{}
		for _, include := range file.Includes {
			if include != nil && !include.Uncertain {
				visit(include.Resolved)
			}
		}
	}
	visit(root)
	return files
}

func safeUnusedIncludeFiles(files map[*File]struct{}) bool {
	for file := range files {
		if file == nil || !file.complete || file.Parsed == nil || file.Parsed.Root == nil || file.Parsed.Broken || file.Parsed.Root.HasError {
			return false
		}
		if len(file.Semantic.UnresolvedReferences()) != 0 || hasActiveIncludeEffects(file) {
			return false
		}
		for _, symbol := range file.Semantic.Symbols {
			switch symbol.Kind {
			case semantic.SymbolFunction:
				if symbol.Decl.Kind != parser.KindFunctionDefinition || !walk.HasChildToken(symbol.Decl, token.KwStock) && file.Walk.Text(symbol.Decl.Field("storage")) != "static" {
					return false
				}
			case semantic.SymbolGlobal:
				if !symbol.Constant {
					return false
				}
			}
		}
	}
	return true
}

func hasActiveIncludeEffects(file *File) bool {
	kinds := []parser.Kind{
		parser.KindDirectiveDefine,
		parser.KindDirectiveUndef,
		parser.KindDirectivePragma,
		parser.KindDirectiveError,
		parser.KindDirectiveWarning,
		parser.KindDirectiveEmit,
		parser.KindDirectiveAssert,
		parser.KindDirectiveLine,
		parser.KindDirectiveFile,
		parser.KindDirectiveEndinput,
		parser.KindDirectiveRaw,
	}
	for _, kind := range kinds {
		for _, node := range file.Walk.OfKind(kind) {
			if !file.Walk.Inactive(node) {
				return true
			}
		}
	}
	return false
}

func (m *Model) includeClosureReferenced(files map[*File]struct{}) bool {
	for _, declarations := range m.Declarations {
		for _, declaration := range declarations {
			if _, included := files[declaration.File]; !included {
				continue
			}
			for _, reference := range m.References(declaration) {
				if _, internal := files[reference.File]; !internal {
					return true
				}
			}
		}
	}
	return false
}
