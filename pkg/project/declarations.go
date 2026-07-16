package project

import (
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/cst"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

type declarationID struct {
	pointer *parser.Node
	source  uint32
	compact syntax.NodeID
}

type referenceID struct {
	declaration declarationID
	pointer     *parser.Node
	source      uint32
	compact     syntax.NodeID
}

type declarationPair struct {
	first  declarationID
	second declarationID
}

type functionVariantKey struct {
	file *File
	name string
}

func (m *Model) References(declaration Declaration) []Reference {
	if m == nil || declaration.File == nil || !declarationSyntax(declaration).Valid() {
		return nil
	}
	return m.references[declarationKey(declaration)]
}

func (m *Model) Resolve(file *File, node *parser.Node) (Declaration, bool) {
	if m == nil || file == nil || node == nil {
		return Declaration{}, false
	}
	return m.resolveSyntax(file, file.syntaxNode(node))
}

func (m *Model) resolveSyntax(file *File, node cst.Node) (Declaration, bool) {
	if m == nil || file == nil || !node.Valid() {
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
	return m.functionVariants(file, file.syntaxNode(node))
}

func (m *Model) functionVariants(file *File, node cst.Node) []Declaration {
	if m == nil || file == nil || !node.Valid() || node.Kind() != parser.KindIdentifier {
		return nil
	}
	name := node.Text()
	key := functionVariantKey{file: file, name: name}
	m.functionVariantsMu.RLock()
	cached, found := m.functionVariantMap[key]
	m.functionVariantsMu.RUnlock()
	if found {
		return cached
	}
	seen := make(map[declarationID]Declaration)
	for _, unit := range m.Units {
		if _, contains := unit.members[file]; !contains {
			continue
		}
		for _, declaration := range m.Declarations[name] {
			if _, included := unit.members[declaration.File]; !included || declaration.Kind != semantic.SymbolFunction || declarationSymbolAmbiguous(declaration) {
				continue
			}
			seen[declarationKey(declaration)] = declaration
		}
	}
	var definitions []Declaration
	var declarations []Declaration
	for _, declaration := range seen {
		if declarationSyntax(declaration).Kind() == parser.KindFunctionDefinition {
			definitions = append(definitions, declaration)
		} else if declarationSyntax(declaration).Kind() == parser.KindFunctionDeclaration {
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
			if !projectStateVariantsCoexist(candidates[left], candidates[right]) {
				candidates = nil
				break
			}
		}
		if candidates == nil {
			break
		}
	}
	m.functionVariantsMu.Lock()
	if cached, found := m.functionVariantMap[key]; found {
		candidates = cached
	} else {
		m.functionVariantMap[key] = candidates
	}
	m.functionVariantsMu.Unlock()
	return candidates
}

func projectStateVariantsCoexist(left, right Declaration) bool {
	leftState := declarationSyntax(left).Field("state").Valid()
	rightState := declarationSyntax(right).Field("state").Valid()
	if !leftState && !rightState {
		return false
	}
	if leftState != rightState {
		return true
	}
	if declarationSymbolStateRaw(left) || declarationSymbolStateRaw(right) {
		return false
	}
	for _, leftName := range declarationSymbolStates(left) {
		for _, rightName := range declarationSymbolStates(right) {
			if leftName == rightName {
				return false
			}
		}
	}
	return true
}

func (m *Model) buildDeclarations() {
	for _, file := range m.Files {
		if file.Semantic != nil {
			for _, symbol := range file.Semantic.Symbols {
				if symbol.Function != nil && symbol.Kind != semantic.SymbolFunction {
					continue
				}
				declaration := Declaration{Name: symbol.Name, Kind: symbol.Kind, File: file, Node: symbol.Decl, Symbol: symbol, syntax: file.Syntax.PointerNode(symbol.Decl)}
				m.Declarations[symbol.Name] = append(m.Declarations[symbol.Name], declaration)
			}
			continue
		}
		for _, symbol := range file.CompactSemantic.Symbols {
			if symbol.Function != syntax.NoNode && symbol.Kind != semantic.SymbolFunction {
				continue
			}
			declaration := Declaration{Name: symbol.Name, Kind: symbol.Kind, File: file, syntax: file.Syntax.CompactNode(symbol.Decl), compactSymbol: symbol}
			m.Declarations[symbol.Name] = append(m.Declarations[symbol.Name], declaration)
		}
	}
	for name := range m.Declarations {
		sortDeclarations(m.Declarations[name])
	}
}

func (m *Model) buildReferences() {
	bySymbol := make(map[*semantic.Symbol]Declaration)
	byCompactSymbol := make(map[*semantic.CompactSymbol]Declaration)
	seen := make(map[referenceID]struct{})
	for _, declarations := range m.Declarations {
		for _, declaration := range declarations {
			if declaration.Symbol != nil {
				bySymbol[declaration.Symbol] = declaration
			} else if declaration.compactSymbol != nil {
				byCompactSymbol[declaration.compactSymbol] = declaration
			}
		}
	}
	for _, file := range m.Files {
		if file.Semantic != nil {
			for _, symbol := range file.Semantic.Symbols {
				declaration, exists := bySymbol[symbol]
				if !exists {
					continue
				}
				for _, reference := range file.Semantic.References(symbol) {
					m.addReference(declaration, Reference{File: file, Node: reference.Node, Kind: reference.Kind, syntax: file.Syntax.PointerNode(reference.Node)}, seen)
				}
			}
			continue
		}
		for _, symbol := range file.CompactSemantic.Symbols {
			declaration, exists := byCompactSymbol[symbol]
			if !exists {
				continue
			}
			for _, reference := range file.CompactSemantic.References(symbol) {
				m.addReference(declaration, Reference{File: file, Kind: reference.Kind, syntax: file.Syntax.CompactNode(reference.Node)}, seen)
			}
		}
	}
	for _, unit := range m.Units {
		for _, file := range unit.Files {
			if file.Semantic != nil {
				for _, reference := range file.Semantic.UnresolvedReferences() {
					m.addUnresolvedReference(unit, file, file.Syntax.PointerNode(reference.Node), reference.Node, reference.Kind, reference.Target, seen)
				}
				continue
			}
			for _, reference := range file.CompactSemantic.UnresolvedReferences() {
				m.addUnresolvedReference(unit, file, file.Syntax.CompactNode(reference.Node), nil, reference.Kind, reference.Target, seen)
			}
		}
	}
	for key := range m.references {
		sort.SliceStable(m.references[key], func(i, j int) bool {
			left, right := m.references[key][i], m.references[key][j]
			if left.File.canonical != right.File.canonical {
				return left.File.canonical < right.File.canonical
			}
			if referenceSyntaxOffset(left) != referenceSyntaxOffset(right) {
				return referenceSyntaxOffset(left) < referenceSyntaxOffset(right)
			}
			return left.Kind < right.Kind
		})
	}
}

func (m *Model) addUnresolvedReference(unit *Unit, file *File, node cst.Node, pointer *parser.Node, kind semantic.ReferenceKind, target semantic.ReferenceTarget, seen map[referenceID]struct{}) {
	if target == semantic.ReferenceFunction {
		variants := m.functionVariants(file, node)
		if len(variants) != 0 {
			for _, declaration := range variants {
				m.addReference(declaration, Reference{File: file, Node: pointer, Kind: kind, syntax: node}, seen)
			}
			return
		}
	}
	declaration, ok := m.resolveInUnit(unit, file, node.Text(), target)
	if ok {
		m.addReference(declaration, Reference{File: file, Node: pointer, Kind: kind, syntax: node}, seen)
	}
}

func (m *Model) resolveInUnit(unit *Unit, from *File, name string, target semantic.ReferenceTarget) (Declaration, bool) {
	var candidates []Declaration
	for _, declaration := range m.Declarations[name] {
		if _, included := unit.members[declaration.File]; !included || declarationSymbolAmbiguous(declaration) {
			continue
		}
		switch target {
		case semantic.ReferenceFunction:
			if declaration.Kind == semantic.SymbolFunction && declarationSyntax(declaration).Kind() == parser.KindFunctionDefinition {
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

func (m *Model) addReference(declaration Declaration, reference Reference, seen map[referenceID]struct{}) {
	key := declarationKey(declaration)
	node := referenceSyntax(reference)
	referenceKey := referenceID{declaration: key, pointer: node.Pointer(), source: reference.File.sourceID, compact: node.ID()}
	if _, exists := seen[referenceKey]; exists {
		return
	}
	seen[referenceKey] = struct{}{}
	m.references[key] = append(m.references[key], reference)
	if m.resolved[reference.File] == nil {
		m.resolved[reference.File] = make(map[cst.Node]Declaration)
	}
	if existing, exists := m.resolved[reference.File][node]; exists && declarationKey(existing) != key {
		delete(m.resolved[reference.File], node)
		if m.ambiguous[reference.File] == nil {
			m.ambiguous[reference.File] = make(map[cst.Node]bool)
		}
		m.ambiguous[reference.File][node] = true
		return
	}
	if m.ambiguous[reference.File][node] {
		return
	}
	m.resolved[reference.File][node] = declaration
}

func declarationSymbolAmbiguous(declaration Declaration) bool {
	if declaration.Symbol != nil {
		return declaration.Symbol.Ambiguous
	}
	return declaration.compactSymbol == nil || declaration.compactSymbol.Ambiguous
}

func declarationSymbolStateRaw(declaration Declaration) bool {
	if declaration.Symbol != nil {
		return declaration.Symbol.StateRaw
	}
	return declaration.compactSymbol != nil && declaration.compactSymbol.StateRaw
}

func declarationSymbolStates(declaration Declaration) []string {
	if declaration.Symbol != nil {
		return declaration.Symbol.States
	}
	if declaration.compactSymbol != nil {
		return declaration.compactSymbol.States
	}
	return nil
}

func declarationNameSyntax(declaration Declaration) cst.Node {
	if declaration.Symbol != nil {
		return declaration.File.Syntax.PointerNode(declaration.Symbol.NameNode)
	}
	if declaration.compactSymbol != nil {
		return declaration.File.Syntax.CompactNode(declaration.compactSymbol.NameNode)
	}
	return declarationSyntax(declaration).Field("name")
}

func declarationSymbolConstant(declaration Declaration) bool {
	if declaration.Symbol != nil {
		return declaration.Symbol.Constant
	}
	return declaration.compactSymbol != nil && declaration.compactSymbol.Constant
}

func declarationSymbolTags(declaration Declaration) []string {
	if declaration.Symbol != nil {
		return declaration.Symbol.Tags
	}
	if declaration.compactSymbol != nil {
		return declaration.compactSymbol.Tags
	}
	return nil
}

func sortDeclarations(declarations []Declaration) {
	sort.SliceStable(declarations, func(i, j int) bool {
		return declarationLess(declarations[i], declarations[j])
	})
}

func declarationKey(declaration Declaration) declarationID {
	if declaration.File == nil {
		return declarationID{}
	}
	node := declarationSyntax(declaration)
	return declarationID{pointer: node.Pointer(), source: declaration.File.sourceID, compact: node.ID()}
}

func declarationSyntax(declaration Declaration) cst.Node {
	if declaration.syntax.Valid() {
		return declaration.syntax
	}
	if declaration.File != nil && declaration.File.Syntax != nil {
		return declaration.File.Syntax.PointerNode(declaration.Node)
	}
	return cst.Node{}
}

func referenceSyntax(reference Reference) cst.Node {
	if reference.syntax.Valid() {
		return reference.syntax
	}
	if reference.File != nil && reference.File.Syntax != nil {
		return reference.File.Syntax.PointerNode(reference.Node)
	}
	return cst.Node{}
}

func declarationSyntaxOffset(declaration Declaration) int {
	return declarationSyntax(declaration).Start()
}

func referenceSyntaxOffset(reference Reference) int {
	return referenceSyntax(reference).Start()
}

func declarationLess(left, right Declaration) bool {
	if left.File.canonical != right.File.canonical {
		return left.File.canonical < right.File.canonical
	}
	if left.File.defines.order != right.File.defines.order {
		return left.File.defines.order < right.File.defines.order
	}
	return declarationSyntaxOffset(left) < declarationSyntaxOffset(right)
}
