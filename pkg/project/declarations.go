package project

import (
	"sort"
	"strconv"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
)

func (m *Model) References(declaration Declaration) []Reference {
	if m == nil || declaration.File == nil || declaration.Node == nil {
		return nil
	}
	return m.references[declarationKey(declaration)]
}

func (m *Model) Resolve(file *File, node *parser.Node) (Declaration, bool) {
	if m == nil || file == nil || node == nil {
		return Declaration{}, false
	}
	declaration, ok := m.resolved[file][node]
	if m.ambiguous[file][node] {
		return Declaration{}, false
	}
	return declaration, ok
}

func (m *Model) FunctionVariants(file *File, node *parser.Node) []Declaration {
	if m == nil || file == nil || node == nil || node.Kind != parser.KindIdentifier {
		return nil
	}
	name := file.Walk.Text(node)
	seen := make(map[string]Declaration)
	for _, unit := range m.Units {
		if _, contains := unit.members[file]; !contains {
			continue
		}
		for _, declaration := range m.Declarations[name] {
			if _, included := unit.members[declaration.File]; !included || declaration.Kind != semantic.SymbolFunction || declaration.Symbol == nil || declaration.Symbol.Ambiguous {
				continue
			}
			seen[declarationKey(declaration)] = declaration
		}
	}
	var definitions []Declaration
	var declarations []Declaration
	for _, declaration := range seen {
		if declaration.Node.Kind == parser.KindFunctionDefinition {
			definitions = append(definitions, declaration)
		} else if declaration.Node.Kind == parser.KindFunctionDeclaration {
			declarations = append(declarations, declaration)
		}
	}
	candidates := definitions
	if len(candidates) == 0 {
		candidates = declarations
	}
	sortDeclarations(candidates)
	for left := range candidates {
		for right := left + 1; right < len(candidates); right++ {
			if !projectStateVariantsCoexist(candidates[left].Symbol, candidates[right].Symbol) {
				return nil
			}
		}
	}
	return candidates
}

func projectStateVariantsCoexist(left, right *semantic.Symbol) bool {
	leftState := left.Decl.Field("state") != nil
	rightState := right.Decl.Field("state") != nil
	if !leftState && !rightState {
		return false
	}
	if leftState != rightState {
		return true
	}
	if left.StateRaw || right.StateRaw {
		return false
	}
	for _, leftName := range left.States {
		for _, rightName := range right.States {
			if leftName == rightName {
				return false
			}
		}
	}
	return true
}

func (m *Model) buildDeclarations() {
	for _, file := range m.Files {
		for _, symbol := range file.Semantic.Symbols {
			if symbol.Function != nil && symbol.Kind != semantic.SymbolFunction {
				continue
			}
			declaration := Declaration{Name: symbol.Name, Kind: symbol.Kind, File: file, Node: symbol.Decl, Symbol: symbol}
			m.Declarations[symbol.Name] = append(m.Declarations[symbol.Name], declaration)
		}
	}
	for name := range m.Declarations {
		sortDeclarations(m.Declarations[name])
	}
}

func (m *Model) buildReferences() {
	bySymbol := make(map[*semantic.Symbol]Declaration)
	seen := make(map[string]struct{})
	for _, declarations := range m.Declarations {
		for _, declaration := range declarations {
			bySymbol[declaration.Symbol] = declaration
		}
	}
	for _, file := range m.Files {
		for _, symbol := range file.Semantic.Symbols {
			declaration, exists := bySymbol[symbol]
			if !exists {
				continue
			}
			for _, reference := range file.Semantic.References(symbol) {
				m.addReference(declaration, Reference{File: file, Node: reference.Node, Kind: reference.Kind}, seen)
			}
		}
	}
	for _, unit := range m.Units {
		for _, file := range unit.Files {
			for _, reference := range file.Semantic.UnresolvedReferences() {
				if reference.Target == semantic.ReferenceFunction {
					variants := m.FunctionVariants(file, reference.Node)
					if len(variants) != 0 {
						for _, declaration := range variants {
							m.addReference(declaration, Reference{File: file, Node: reference.Node, Kind: reference.Kind}, seen)
						}
						continue
					}
				}
				name := file.Walk.Text(reference.Node)
				declaration, ok := m.resolveInUnit(unit, file, name, reference.Target)
				if !ok {
					continue
				}
				m.addReference(declaration, Reference{File: file, Node: reference.Node, Kind: reference.Kind}, seen)
			}
		}
	}
	for key := range m.references {
		sort.SliceStable(m.references[key], func(i, j int) bool {
			left, right := m.references[key][i], m.references[key][j]
			if left.File.canonical != right.File.canonical {
				return left.File.canonical < right.File.canonical
			}
			if left.Node.Start != right.Node.Start {
				return left.Node.Start < right.Node.Start
			}
			return left.Kind < right.Kind
		})
	}
}

func (m *Model) resolveInUnit(unit *Unit, from *File, name string, target semantic.ReferenceTarget) (Declaration, bool) {
	var candidates []Declaration
	for _, declaration := range m.Declarations[name] {
		if _, included := unit.members[declaration.File]; !included || declaration.Symbol == nil || declaration.Symbol.Ambiguous {
			continue
		}
		switch target {
		case semantic.ReferenceFunction:
			if declaration.Kind == semantic.SymbolFunction && declaration.Node.Kind == parser.KindFunctionDefinition {
				candidates = append(candidates, declaration)
			}
		case semantic.ReferenceValue:
			if declaration.File == from {
				return Declaration{}, false
			}
			if declaration.Kind == semantic.SymbolGlobal || declaration.Kind == semantic.SymbolEnumRoot || declaration.Kind == semantic.SymbolEnumEntry {
				candidates = append(candidates, declaration)
			}
		}
	}
	if len(candidates) != 1 {
		return Declaration{}, false
	}
	return candidates[0], true
}

func (m *Model) addReference(declaration Declaration, reference Reference, seen map[string]struct{}) {
	key := declarationKey(declaration)
	referenceKey := key + "\x00" + reference.File.instance + "\x00" + strconv.Itoa(reference.Node.Start)
	if _, exists := seen[referenceKey]; exists {
		return
	}
	seen[referenceKey] = struct{}{}
	m.references[key] = append(m.references[key], reference)
	if m.resolved[reference.File] == nil {
		m.resolved[reference.File] = make(map[*parser.Node]Declaration)
	}
	if existing, exists := m.resolved[reference.File][reference.Node]; exists && declarationKey(existing) != key {
		delete(m.resolved[reference.File], reference.Node)
		if m.ambiguous[reference.File] == nil {
			m.ambiguous[reference.File] = make(map[*parser.Node]bool)
		}
		m.ambiguous[reference.File][reference.Node] = true
		return
	}
	if m.ambiguous[reference.File][reference.Node] {
		return
	}
	m.resolved[reference.File][reference.Node] = declaration
}

func sortDeclarations(declarations []Declaration) {
	sort.SliceStable(declarations, func(i, j int) bool {
		if declarations[i].File.instance != declarations[j].File.instance {
			return declarations[i].File.instance < declarations[j].File.instance
		}
		return declarations[i].Node.Start < declarations[j].Node.Start
	})
}

func declarationKey(declaration Declaration) string {
	return declaration.File.instance + "\x00" + strconv.Itoa(declaration.Node.Start)
}
