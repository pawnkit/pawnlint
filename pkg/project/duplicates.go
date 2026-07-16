package project

import (
	"sort"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/cst"
)

func (m *Model) DuplicateFunctions() []DuplicateFunction {
	if m == nil {
		return nil
	}
	return m.duplicateFunctions
}

func (m *Model) buildDuplicateFunctions() []DuplicateFunction {
	seen := make(map[declarationPair]struct{})
	var result []DuplicateFunction
	for _, unit := range m.Units {
		macroQualifiers := functionMacroQualifiers(unit)
		byName := make(map[string][]Declaration)
		for _, file := range unit.Files {
			for _, node := range file.Syntax.OfKind(parser.KindFunctionDefinition) {
				if file.Syntax.Uncertain(node) || node.Field("state").Valid() || node.Field("generic").Valid() || insideErroredDeclaration(file, node) || insideFunction(file, node) {
					continue
				}
				storage := node.Field("storage").Text()
				if storage == "hook" {
					continue
				}
				if storage != "" && storage != "stock" && storage != "static" && storage != "public" {
					continue
				}
				tag := strings.TrimSuffix(node.Field("tag").Text(), ":")
				if _, exists := macroQualifiers[tag]; exists {
					continue
				}
				nameNode := node.Field("name")
				name := nameNode.Text()
				if name == "" || strings.HasPrefix(name, "operator") {
					continue
				}
				key := name
				if storage == "public" {
					key = file.canonical + "\x00" + name
				}
				byName[key] = append(byName[key], Declaration{Name: name, Kind: semantic.SymbolFunction, File: file, Node: node.Pointer(), syntax: node})
			}
		}
		for _, declarations := range byName {
			if len(declarations) < 2 {
				continue
			}
			name := declarations[0].Name
			for _, duplicate := range declarations[1:] {
				first := declarations[0]
				if first.File.canonical == duplicate.File.canonical && declarationSyntaxOffset(first) == declarationSyntaxOffset(duplicate) {
					continue
				}
				key := declarationPair{first: declarationKey(first), second: declarationKey(duplicate)}
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				owner := duplicate.File
				if !owner.Provided {
					owner = unit.Root
				}
				result = append(result, DuplicateFunction{Name: name, First: first, Second: duplicate, Owner: owner})
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
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return declarationLess(result[i].First, result[j].First)
	})
	return result
}

func (m *Model) DuplicateGlobals() []DuplicateGlobal {
	if m == nil {
		return nil
	}
	return m.duplicateGlobals
}

func (m *Model) buildDuplicateGlobals() []DuplicateGlobal {
	seen := make(map[declarationPair]struct{})
	var result []DuplicateGlobal
	for _, unit := range m.Units {
		byName := make(map[string][]Declaration)
		for _, file := range unit.Files {
			for _, declaration := range declarationsForFile(file) {
				if declaration.File != file || declaration.Kind != semantic.SymbolGlobal {
					continue
				}
				if numericSeparatorArtifact(declaration) {
					continue
				}
				node := declarationSyntax(declaration)
				parent := file.Syntax.Parent(node)
				if node.Field("state").Valid() || parent.Valid() && (parent.Field("state").Valid() || parent.HasError() || file.Syntax.Uncertain(parent)) {
					continue
				}
				storage := parent.Field("storage").Text()
				if storage != "new" && storage != "static" && storage != "const" {
					continue
				}
				key := declaration.Name
				if storage == "static" {
					key = file.canonical + "\x00" + declaration.Name
				}
				byName[key] = append(byName[key], declaration)
			}
		}
		for _, declarations := range byName {
			if len(declarations) < 2 {
				continue
			}
			name := declarations[0].Name
			for _, duplicate := range declarations[1:] {
				first := declarations[0]
				if first.File.canonical == duplicate.File.canonical && declarationSyntaxOffset(first) == declarationSyntaxOffset(duplicate) {
					continue
				}
				key := declarationPair{first: declarationKey(first), second: declarationKey(duplicate)}
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				owner := duplicate.File
				if !owner.Provided {
					owner = unit.Root
				}
				result = append(result, DuplicateGlobal{Name: name, First: first, Second: duplicate, Owner: owner})
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

func insideErroredDeclaration(file *File, node cst.Node) bool {
	for parent := file.Syntax.Parent(node); parent.Valid() && parent.Kind() != parser.KindSourceFile; parent = file.Syntax.Parent(parent) {
		if parent.Kind() == parser.KindVariableDeclaration && parent.HasError() {
			return true
		}
	}
	return false
}

func insideFunction(file *File, node cst.Node) bool {
	for parent := file.Syntax.Parent(node); parent.Valid(); parent = file.Syntax.Parent(parent) {
		if parent.Kind() == parser.KindFunctionDefinition {
			return true
		}
	}
	return false
}

func functionMacroQualifiers(unit *Unit) map[string]struct{} {
	qualifiers := make(map[string]struct{})
	for _, file := range unit.Files {
		for index := 0; index+3 < file.Syntax.TokenCount(); index++ {
			hash := file.Syntax.Token(index)
			directive := file.Syntax.Token(index + 1)
			name := file.Syntax.Token(index + 2)
			colon := file.Syntax.Token(index + 3)
			if hash.Kind() == token.Hash && directive.Kind() == token.Identifier && directive.Text() == "define" && name.Kind() == token.Identifier && colon.Kind() == token.Colon {
				qualifiers[name.Text()] = struct{}{}
			}
		}
	}
	return qualifiers
}

func numericSeparatorArtifact(declaration Declaration) bool {
	file := declaration.File
	name := declarationNameSyntax(declaration)
	if file == nil || !name.Valid() {
		return false
	}
	for index := 0; index < file.Syntax.TokenCount(); index++ {
		current := file.Syntax.Token(index)
		if current.Start() != name.Start() || index == 0 {
			continue
		}
		previous := file.Syntax.Token(index - 1)
		return current.Kind() == token.Identifier && strings.HasPrefix(declaration.Name, "_") && previous.Kind() == token.IntLiteral && previous.End() == current.Start()
	}
	return false
}
