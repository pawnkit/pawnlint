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
		if declarationSyntaxOffset(result[i].Second) != declarationSyntaxOffset(result[j].Second) {
			return declarationSyntaxOffset(result[i].Second) < declarationSyntaxOffset(result[j].Second)
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func conflictEligible(declaration Declaration, qualifiers map[string]struct{}) bool {
	file := declaration.File
	node := declarationSyntax(declaration)
	if file == nil || declarationSymbolAmbiguous(declaration) || !node.Valid() || node.HasError() {
		return false
	}
	switch declaration.Kind {
	case semantic.SymbolFunction:
		if node.Kind() != parser.KindFunctionDefinition || node.Field("state").Valid() || node.Field("generic").Valid() || insideErroredDeclaration(file, node) || insideFunction(file, node) {
			return false
		}
		storage := node.Field("storage").Text()
		if storage == "hook" || storage == "public" || storage != "" && storage != "stock" && storage != "static" {
			return false
		}
		tag := strings.TrimSuffix(node.Field("tag").Text(), ":")
		_, macroQualified := qualifiers[tag]
		return !macroQualified
	case semantic.SymbolGlobal:
		if numericSeparatorArtifact(declaration) {
			return false
		}
		parent := file.Syntax.Parent(node)
		if !parent.Valid() || node.Field("state").Valid() || parent.Field("state").Valid() || parent.HasError() || file.Syntax.Uncertain(parent) {
			return false
		}
		storage := parent.Field("storage").Text()
		return storage == "new" || storage == "const"
	case semantic.SymbolEnumRoot, semantic.SymbolEnumEntry:
		return !file.Syntax.Uncertain(node)
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
