package project

import (
	"sort"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
)

func (m *Model) DuplicateFunctions() []DuplicateFunction {
	if m == nil {
		return nil
	}
	seen := make(map[declarationPair]struct{})
	var result []DuplicateFunction
	for _, unit := range m.Units {
		macroQualifiers := functionMacroQualifiers(unit)
		byName := make(map[string][]Declaration)
		for _, file := range unit.Files {
			for _, node := range file.Walk.OfKind(parser.KindFunctionDefinition) {
				if file.Walk.Uncertain(node) || node.Field("state") != nil || node.Field("generic") != nil || insideErroredDeclaration(file, node) || insideFunction(file, node) {
					continue
				}
				storage := file.Walk.Text(node.Field("storage"))
				if storage == "hook" {
					continue
				}
				if storage != "" && storage != "stock" && storage != "static" && storage != "public" {
					continue
				}
				tag := strings.TrimSuffix(file.Walk.Text(node.Field("tag")), ":")
				if _, exists := macroQualifiers[tag]; exists {
					continue
				}
				nameNode := node.Field("name")
				name := file.Walk.Text(nameNode)
				if name == "" || strings.HasPrefix(name, "operator") {
					continue
				}
				key := name
				if storage == "public" {
					key = file.canonical + "\x00" + name
				}
				byName[key] = append(byName[key], Declaration{Name: name, Kind: semantic.SymbolFunction, File: file, Node: node})
			}
		}
		for _, declarations := range byName {
			if len(declarations) < 2 {
				continue
			}
			name := declarations[0].Name
			for _, duplicate := range declarations[1:] {
				first := declarations[0]
				if first.File.canonical == duplicate.File.canonical && first.Node.Start == duplicate.Node.Start {
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
		if result[i].Second.Node.Start != result[j].Second.Node.Start {
			return result[i].Second.Node.Start < result[j].Second.Node.Start
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
	seen := make(map[declarationPair]struct{})
	var result []DuplicateGlobal
	for _, unit := range m.Units {
		byName := make(map[string][]Declaration)
		for _, file := range unit.Files {
			for _, symbol := range file.Semantic.Symbols {
				if symbol.Kind != semantic.SymbolGlobal {
					continue
				}
				if numericSeparatorArtifact(file, symbol) {
					continue
				}
				parent := file.Walk.Parent(symbol.Decl)
				if symbol.Decl.Field("state") != nil || parent != nil && (parent.Field("state") != nil || parent.HasError || file.Walk.Uncertain(parent)) {
					continue
				}
				storage := file.Walk.Text(parent.Field("storage"))
				if storage != "new" && storage != "static" && storage != "const" {
					continue
				}
				key := symbol.Name
				if storage == "static" {
					key = file.canonical + "\x00" + symbol.Name
				}
				byName[key] = append(byName[key], Declaration{
					Name: symbol.Name, Kind: symbol.Kind, File: file, Node: symbol.Decl, Symbol: symbol,
				})
			}
		}
		for _, declarations := range byName {
			if len(declarations) < 2 {
				continue
			}
			name := declarations[0].Name
			for _, duplicate := range declarations[1:] {
				first := declarations[0]
				if first.File.canonical == duplicate.File.canonical && first.Node.Start == duplicate.Node.Start {
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
		if result[i].Second.Node.Start != result[j].Second.Node.Start {
			return result[i].Second.Node.Start < result[j].Second.Node.Start
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func insideErroredDeclaration(file *File, node *parser.Node) bool {
	for parent := file.Walk.Parent(node); parent != nil && parent.Kind != parser.KindSourceFile; parent = file.Walk.Parent(parent) {
		if parent.Kind == parser.KindVariableDeclaration && parent.HasError {
			return true
		}
	}
	return false
}

func insideFunction(file *File, node *parser.Node) bool {
	for parent := file.Walk.Parent(node); parent != nil; parent = file.Walk.Parent(parent) {
		if parent.Kind == parser.KindFunctionDefinition {
			return true
		}
	}
	return false
}

func functionMacroQualifiers(unit *Unit) map[string]struct{} {
	qualifiers := make(map[string]struct{})
	for _, file := range unit.Files {
		for index := 0; index+3 < len(file.Parsed.Tokens); index++ {
			hash := file.Parsed.Tokens[index]
			directive := file.Parsed.Tokens[index+1]
			name := file.Parsed.Tokens[index+2]
			colon := file.Parsed.Tokens[index+3]
			if hash.Kind == token.Hash && directive.Kind == token.Identifier && directive.Text(file.Source) == "define" && name.Kind == token.Identifier && colon.Kind == token.Colon {
				qualifiers[name.Text(file.Source)] = struct{}{}
			}
		}
	}
	return qualifiers
}

func numericSeparatorArtifact(file *File, symbol *semantic.Symbol) bool {
	if symbol == nil || symbol.NameNode == nil || file == nil || file.Parsed == nil {
		return false
	}
	for index := range file.Parsed.Tokens {
		current := &file.Parsed.Tokens[index]
		if current.Start.Offset != symbol.NameNode.Start || index == 0 {
			continue
		}
		previous := &file.Parsed.Tokens[index-1]
		return current.Kind == token.Identifier && strings.HasPrefix(symbol.Name, "_") && previous.Kind == token.IntLiteral && previous.End.Offset == current.Start.Offset
	}
	return false
}
