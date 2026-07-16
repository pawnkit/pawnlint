package project

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/syntax"
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
		if file == nil || !file.complete || fileHasParseErrors(file) {
			return false
		}
		if fileUnresolvedReferenceCount(file) != 0 || hasActiveIncludeEffects(file) {
			return false
		}
		if !visitDeclarationsForFile(file, func(declaration Declaration) bool {
			switch declaration.Kind {
			case semantic.SymbolFunction:
				node := declarationSyntax(declaration)
				if node.Kind() != parser.KindFunctionDefinition || !node.HasChildToken(token.KwStock) && node.Field("storage").Text() != "static" {
					return false
				}
			case semantic.SymbolGlobal:
				if !declarationSymbolConstant(declaration) {
					return false
				}
			}
			return true
		}) {
			return false
		}
	}
	return true
}

func visitDeclarationsForFile(file *File, visit func(Declaration) bool) bool {
	if file.Semantic != nil {
		for _, symbol := range file.Semantic.Symbols {
			if symbol.Function != nil && symbol.Kind != semantic.SymbolFunction {
				continue
			}
			if !visit(Declaration{Name: symbol.Name, Kind: symbol.Kind, File: file, Node: symbol.Decl, Symbol: symbol, syntax: file.Syntax.PointerNode(symbol.Decl)}) {
				return false
			}
		}
		return true
	}
	for _, symbol := range file.CompactSemantic.Symbols {
		if symbol.Function != syntax.NoNode && symbol.Kind != semantic.SymbolFunction {
			continue
		}
		if !visit(Declaration{Name: symbol.Name, Kind: symbol.Kind, File: file, compactSymbol: symbol, syntax: file.Syntax.CompactNode(symbol.Decl)}) {
			return false
		}
	}
	return true
}

func fileHasParseErrors(file *File) bool {
	if file.Parsed != nil {
		return file.Parsed.HasParseErrors()
	}
	return file.CompactParsed == nil || file.CompactParsed.HasParseErrors()
}

func fileUnresolvedReferenceCount(file *File) int {
	if file.Semantic != nil {
		return len(file.Semantic.UnresolvedReferences())
	}
	if file.CompactSemantic != nil {
		return len(file.CompactSemantic.UnresolvedReferences())
	}
	return 0
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
		for _, node := range file.Syntax.OfKind(kind) {
			if !file.Syntax.Inactive(node) {
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
