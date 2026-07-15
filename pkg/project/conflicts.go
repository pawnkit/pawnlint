package project

import (
	"sort"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
)

type SymbolConflict struct {
	Name   string
	First  Declaration
	Second Declaration
	Owner  *File
}

func (m *Model) ConflictingIncludeSymbols() []SymbolConflict {
	if m == nil {
		return nil
	}
	return append([]SymbolConflict(nil), m.symbolConflicts...)
}

func (m *Model) buildConflictingIncludeSymbols() []SymbolConflict {
	seen := make(map[declarationPair]struct{})
	var result []SymbolConflict
	for _, unit := range m.Units {
		qualifiers := functionMacroQualifiers(unit)
		byName := make(map[string][]Declaration)
		for name, declarations := range m.Declarations {
			for _, declaration := range declarations {
				if _, included := unit.members[declaration.File]; !included || !conflictEligible(declaration, qualifiers) {
					continue
				}
				byName[name] = append(byName[name], declaration)
			}
		}
		for name, declarations := range byName {
			sortDeclarations(declarations)
			for index := 1; index < len(declarations); index++ {
				second := declarations[index]
				for firstIndex := 0; firstIndex < index; firstIndex++ {
					first := declarations[firstIndex]
					if first.File.canonical == second.File.canonical || duplicateRuleCovers(first, second) {
						continue
					}
					if first.File == unit.Root && second.File == unit.Root {
						continue
					}
					key := declarationPair{first: declarationKey(first), second: declarationKey(second)}
					if _, exists := seen[key]; exists {
						break
					}
					seen[key] = struct{}{}
					result = append(result, SymbolConflict{Name: name, First: first, Second: second, Owner: unit.Root})
					break
				}
			}
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Second.File.canonical != result[j].Second.File.canonical {
			return result[i].Second.File.canonical < result[j].Second.File.canonical
		}
		if result[i].Second.Node.Start != result[j].Second.Node.Start {
			return result[i].Second.Node.Start < result[j].Second.Node.Start
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func conflictEligible(declaration Declaration, qualifiers map[string]struct{}) bool {
	file := declaration.File
	symbol := declaration.Symbol
	if file == nil || symbol == nil || symbol.Ambiguous || declaration.Node == nil || declaration.Node.HasError {
		return false
	}
	switch declaration.Kind {
	case semantic.SymbolFunction:
		if declaration.Node.Kind != parser.KindFunctionDefinition || declaration.Node.Field("state") != nil || declaration.Node.Field("generic") != nil || insideErroredDeclaration(file, declaration.Node) || insideFunction(file, declaration.Node) {
			return false
		}
		storage := file.Walk.Text(declaration.Node.Field("storage"))
		if storage == "hook" || storage == "public" || storage != "" && storage != "stock" && storage != "static" {
			return false
		}
		tag := strings.TrimSuffix(file.Walk.Text(declaration.Node.Field("tag")), ":")
		_, macroQualified := qualifiers[tag]
		return !macroQualified
	case semantic.SymbolGlobal:
		if numericSeparatorArtifact(file, symbol) {
			return false
		}
		parent := file.Walk.Parent(symbol.Decl)
		if parent == nil || symbol.Decl.Field("state") != nil || parent.Field("state") != nil || parent.HasError || file.Walk.Uncertain(parent) {
			return false
		}
		storage := file.Walk.Text(parent.Field("storage"))
		return storage == "new" || storage == "const"
	case semantic.SymbolEnumRoot, semantic.SymbolEnumEntry:
		return !file.Walk.Uncertain(symbol.Decl)
	default:
		return false
	}
}

func duplicateRuleCovers(first, second Declaration) bool {
	return first.Kind == semantic.SymbolFunction && second.Kind == semantic.SymbolFunction ||
		first.Kind == semantic.SymbolGlobal && second.Kind == semantic.SymbolGlobal ||
		enumSymbol(first.Kind) && enumSymbol(second.Kind)
}

func enumSymbol(kind semantic.SymbolKind) bool {
	return kind == semantic.SymbolEnumRoot || kind == semantic.SymbolEnumEntry
}
