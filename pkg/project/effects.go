package project

import (
	"sort"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/cst"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

type FunctionEffects struct {
	Complete          bool
	Pure              bool
	ReadsGlobals      []Declaration
	WritesGlobals     []Declaration
	MutatedParameters []int
	Calls             []Declaration
}

type functionEffectState struct {
	complete            bool
	intrinsicImpure     bool
	pure                bool
	reads               map[declarationID]Declaration
	writes              map[declarationID]Declaration
	mutated             map[int]bool
	calls               []Call
	parameterIndexes    map[cst.Node]int
	referenceParameters map[cst.Node]bool
}

func (m *Model) FunctionEffects(function Declaration) (FunctionEffects, bool) {
	if m == nil || !m.functionEffects {
		return FunctionEffects{}, false
	}
	m.effectsOnce.Do(m.buildFunctionEffects)
	effects, ok := m.effects[declarationKey(function)]
	if !ok {
		return FunctionEffects{}, false
	}
	effects.ReadsGlobals = append([]Declaration(nil), effects.ReadsGlobals...)
	effects.WritesGlobals = append([]Declaration(nil), effects.WritesGlobals...)
	effects.MutatedParameters = append([]int(nil), effects.MutatedParameters...)
	effects.Calls = append([]Declaration(nil), effects.Calls...)
	return effects, true
}

func (m *Model) buildFunctionEffects() {
	m.effects = make(map[declarationID]FunctionEffects)
	if m.CallGraph == nil {
		return
	}
	states := make(map[declarationID]*functionEffectState, len(m.CallGraph.Functions))
	pointerReferences := make(map[*File]map[*parser.Node]semantic.ReferenceKind)
	pointerSymbols := make(map[*parser.Node][]*semantic.Symbol)
	compactReferences := make(map[*File]map[syntax.NodeID]semantic.ReferenceKind)
	compactSymbols := make(map[syntax.NodeID][]*semantic.CompactSymbol)
	for _, file := range m.Files {
		if file.Semantic != nil {
			pointerReferences[file] = pointerEffectReferenceKinds(file)
			for _, symbol := range file.Semantic.Symbols {
				if symbol.Function != nil {
					pointerSymbols[symbol.Function] = append(pointerSymbols[symbol.Function], symbol)
				}
			}
			continue
		}
		compactReferences[file] = compactEffectReferenceKinds(file)
		for _, symbol := range file.CompactSemantic.Symbols {
			if symbol.Function != syntax.NoNode {
				compactSymbols[symbol.Function] = append(compactSymbols[symbol.Function], symbol)
			}
		}
	}
	for _, function := range m.CallGraph.Functions {
		if function.Node != nil {
			states[declarationKey(function)] = m.directPointerFunctionEffects(function, pointerReferences[function.File], pointerSymbols[function.Node])
		} else {
			states[declarationKey(function)] = m.directCompactFunctionEffects(function, compactReferences[function.File], compactSymbols[declarationSyntax(function).ID()])
		}
	}
	for iteration := 0; iteration <= len(states); iteration++ {
		changed := false
		for _, function := range m.CallGraph.Functions {
			state := states[declarationKey(function)]
			if m.mergeFunctionEffects(function, state, states) {
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	for _, function := range m.CallGraph.Functions {
		state := states[declarationKey(function)]
		m.effects[declarationKey(function)] = publicFunctionEffects(state)
	}
}

func (m *Model) directPointerFunctionEffects(function Declaration, referenceKinds map[*parser.Node]semantic.ReferenceKind, symbols []*semantic.Symbol) *functionEffectState {
	state := &functionEffectState{
		complete:            true,
		pure:                true,
		reads:               make(map[declarationID]Declaration),
		writes:              make(map[declarationID]Declaration),
		mutated:             make(map[int]bool),
		parameterIndexes:    make(map[cst.Node]int),
		referenceParameters: make(map[cst.Node]bool),
	}
	file := function.File
	node := function.Node
	if file == nil || node == nil || node.HasError || file.Walk.Inactive(node) || file.Walk.Uncertain(node) {
		state.complete = false
		state.intrinsicImpure = true
		state.pure = false
		return state
	}
	m.indexPointerEffectParameters(file, node, state)
	for _, symbol := range symbols {
		if symbol.Kind != semantic.SymbolLocal || !walk.HasChildToken(file.Walk.Parent(symbol.Decl), token.KwStatic) {
			continue
		}
		state.intrinsicImpure = true
	}
	state.calls = append([]Call(nil), m.CallGraph.Outgoing(function)...)
	knownCalls := make(map[*parser.Node]bool)
	for _, call := range state.calls {
		knownCalls[call.Node] = true
	}
	var visit func(*parser.Node)
	visit = func(current *parser.Node) {
		if current == nil || file.Walk.Inactive(current) {
			return
		}
		if current != node && current.Kind == parser.KindFunctionDefinition {
			state.complete = false
			return
		}
		if file.Walk.Uncertain(current) {
			state.complete = false
			return
		}
		switch current.Kind {
		case parser.KindMacroInvocation, parser.KindMacroInvocationBlock, parser.KindConditionalSplice:
			state.complete = false
		case parser.KindStateStatement:
			state.intrinsicImpure = true
		case parser.KindCallExpression:
			if !knownCalls[current] {
				state.complete = false
			}
		case parser.KindIdentifier:
			m.applyDirectPointerReferenceEffect(state, file, current, referenceKinds)
		}
		for _, child := range current.Children {
			visit(child)
		}
	}
	visit(node)
	state.pure = state.complete && !state.intrinsicImpure && len(state.reads) == 0 && len(state.writes) == 0 && len(state.mutated) == 0
	return state
}

func (m *Model) applyDirectPointerReferenceEffect(state *functionEffectState, file *File, identifier *parser.Node, referenceKinds map[*parser.Node]semantic.ReferenceKind) {
	kind, reference := referenceKinds[identifier]
	if !reference {
		return
	}
	symbol := file.Semantic.Resolve(identifier)
	if symbol != nil && symbol.Kind == semantic.SymbolParameter {
		key := file.Syntax.PointerNode(symbol.Decl)
		if state.referenceParameters[key] && kind != semantic.ReferenceRead {
			state.mutated[state.parameterIndexes[key]] = true
		}
	}
	declaration, ok := m.Resolve(file, identifier)
	if !ok || declaration.Kind != semantic.SymbolGlobal {
		return
	}
	if kind == semantic.ReferenceRead && declarationSymbolConstant(declaration) {
		return
	}
	key := declarationKey(declaration)
	if kind == semantic.ReferenceRead {
		state.reads[key] = declaration
	} else {
		state.writes[key] = declaration
	}
}

func (m *Model) indexPointerEffectParameters(file *File, function *parser.Node, state *functionEffectState) {
	list := function.Field("parameters")
	if list == nil {
		return
	}
	index := 0
	for _, parameter := range list.Children {
		if parameter.Kind != parser.KindParameter {
			continue
		}
		for _, symbol := range file.Semantic.Symbols {
			if symbol.Kind != semantic.SymbolParameter || symbol.Decl != parameter || symbol.Ambiguous {
				continue
			}
			key := file.Syntax.PointerNode(symbol.Decl)
			state.parameterIndexes[key] = index
			state.referenceParameters[key] = walk.ReferencesByAmpersand(file.Parsed.Tokens, parameter) || pointerEffectHasDimension(parameter)
			break
		}
		index++
	}
}

func (m *Model) directCompactFunctionEffects(function Declaration, referenceKinds map[syntax.NodeID]semantic.ReferenceKind, symbols []*semantic.CompactSymbol) *functionEffectState {
	state := &functionEffectState{
		complete:            true,
		pure:                true,
		reads:               make(map[declarationID]Declaration),
		writes:              make(map[declarationID]Declaration),
		mutated:             make(map[int]bool),
		parameterIndexes:    make(map[cst.Node]int),
		referenceParameters: make(map[cst.Node]bool),
	}
	file := function.File
	if file == nil {
		state.complete = false
		state.intrinsicImpure = true
		state.pure = false
		return state
	}
	node := declarationSyntax(function).ID()
	tree := file.CompactWalk
	if tree == nil || !tree.Tree.Valid(node) || tree.Tree.HasError(node) || tree.Inactive(node) || tree.Uncertain(node) {
		state.complete = false
		state.intrinsicImpure = true
		state.pure = false
		return state
	}
	m.indexCompactEffectParameters(file, node, state)
	for _, symbol := range symbols {
		if symbol.Kind != semantic.SymbolLocal || !compactHasChildToken(tree, tree.Parent(symbol.Decl), token.KwStatic) {
			continue
		}
		state.intrinsicImpure = true
	}
	state.calls = append([]Call(nil), m.CallGraph.Outgoing(function)...)
	knownCalls := make(map[syntax.NodeID]bool)
	for _, call := range state.calls {
		knownCalls[callSyntax(call).ID()] = true
	}
	var visit func(syntax.NodeID)
	visit = func(current syntax.NodeID) {
		if !tree.Tree.Valid(current) || tree.Inactive(current) {
			return
		}
		if current != node && tree.Tree.Kind(current) == parser.KindFunctionDefinition {
			state.complete = false
			return
		}
		if tree.Uncertain(current) {
			state.complete = false
			return
		}
		switch tree.Tree.Kind(current) {
		case parser.KindMacroInvocation, parser.KindMacroInvocationBlock, parser.KindConditionalSplice:
			state.complete = false
		case parser.KindStateStatement:
			state.intrinsicImpure = true
		case parser.KindCallExpression:
			if !knownCalls[current] {
				state.complete = false
			}
		case parser.KindIdentifier:
			m.applyDirectCompactReferenceEffect(state, file, current, referenceKinds)
		}
		for index := 0; index < tree.Tree.ChildCount(current); index++ {
			visit(tree.Tree.Child(current, index))
		}
	}
	visit(node)
	state.pure = state.complete && !state.intrinsicImpure && len(state.reads) == 0 && len(state.writes) == 0 && len(state.mutated) == 0
	return state
}

func (m *Model) applyDirectCompactReferenceEffect(state *functionEffectState, file *File, identifier syntax.NodeID, referenceKinds map[syntax.NodeID]semantic.ReferenceKind) {
	kind, reference := referenceKinds[identifier]
	if !reference {
		return
	}
	symbol := file.CompactSemantic.Resolve(identifier)
	if symbol != nil && symbol.Kind == semantic.SymbolParameter {
		key := file.Syntax.CompactNode(symbol.Decl)
		if state.referenceParameters[key] && kind != semantic.ReferenceRead {
			state.mutated[state.parameterIndexes[key]] = true
		}
	}
	declaration, ok := m.resolveSyntax(file, file.Syntax.CompactNode(identifier))
	if !ok || declaration.Kind != semantic.SymbolGlobal {
		return
	}
	if kind == semantic.ReferenceRead && declarationSymbolConstant(declaration) {
		return
	}
	key := declarationKey(declaration)
	if kind == semantic.ReferenceRead {
		state.reads[key] = declaration
	} else {
		state.writes[key] = declaration
	}
}

func (m *Model) indexCompactEffectParameters(file *File, function syntax.NodeID, state *functionEffectState) {
	tree := file.CompactWalk.Tree
	list := tree.Field(function, "parameters")
	if !tree.Valid(list) {
		return
	}
	index := 0
	for child := 0; child < tree.ChildCount(list); child++ {
		parameter := tree.Child(list, child)
		if tree.Kind(parameter) != parser.KindParameter {
			continue
		}
		for _, symbol := range file.CompactSemantic.Symbols {
			if symbol.Kind != semantic.SymbolParameter || symbol.Decl != parameter || symbol.Ambiguous {
				continue
			}
			key := file.Syntax.CompactNode(symbol.Decl)
			state.parameterIndexes[key] = index
			state.referenceParameters[key] = effectReferencesByAmpersand(file, file.Syntax.CompactNode(parameter)) || compactEffectHasDimension(tree, parameter)
			break
		}
		index++
	}
}

func (m *Model) mergeFunctionEffects(function Declaration, state *functionEffectState, states map[declarationID]*functionEffectState) bool {
	complete := state.complete
	intrinsicImpure := state.intrinsicImpure
	reads := cloneEffectDeclarations(state.reads)
	writes := cloneEffectDeclarations(state.writes)
	mutated := cloneEffectParameters(state.mutated)
	for _, call := range state.calls {
		callee := states[declarationKey(call.Callee)]
		if callee == nil || !callee.complete {
			complete = false
			continue
		}
		intrinsicImpure = intrinsicImpure || callee.intrinsicImpure
		mergeEffectDeclarations(reads, callee.reads)
		mergeEffectDeclarations(writes, callee.writes)
		if len(callee.mutated) != 0 && !m.mapMutatedArguments(state, call, callee.mutated, mutated, writes) {
			complete = false
		}
	}
	pure := complete && !intrinsicImpure && len(reads) == 0 && len(writes) == 0 && len(mutated) == 0
	changed := complete != state.complete || intrinsicImpure != state.intrinsicImpure || pure != state.pure || !sameEffectDeclarations(reads, state.reads) || !sameEffectDeclarations(writes, state.writes) || !sameEffectParameters(mutated, state.mutated)
	state.complete = complete
	state.intrinsicImpure = intrinsicImpure
	state.pure = pure
	state.reads = reads
	state.writes = writes
	state.mutated = mutated
	return changed
}

func (m *Model) mapMutatedArguments(state *functionEffectState, call Call, calleeMutated map[int]bool, mutated map[int]bool, writes map[declarationID]Declaration) bool {
	arguments := callSyntax(call).Field("arguments")
	if !arguments.Valid() {
		return false
	}
	complete := true
	for index := range calleeMutated {
		argumentIndex := index + call.ArgumentOffset
		if argumentIndex < 0 || argumentIndex >= arguments.ChildCount() {
			complete = false
			continue
		}
		identifier := effectBaseIdentifier(arguments.Child(argumentIndex))
		if !identifier.Valid() {
			complete = false
			continue
		}
		if symbol, _, _ := effectResolvedSymbol(call.File, identifier); symbol.Valid() {
			if parameterIndex, ok := state.parameterIndexes[symbol]; ok && state.referenceParameters[symbol] {
				mutated[parameterIndex] = true
			}
			continue
		}
		if declaration, ok := m.resolveSyntax(call.File, identifier); ok && declaration.Kind == semantic.SymbolGlobal {
			writes[declarationKey(declaration)] = declaration
			continue
		}
		complete = false
	}
	return complete
}

func pointerEffectReferenceKinds(file *File) map[*parser.Node]semantic.ReferenceKind {
	result := make(map[*parser.Node]semantic.ReferenceKind)
	for _, symbol := range file.Semantic.Symbols {
		for _, reference := range file.Semantic.References(symbol) {
			result[reference.Node] = pointerEffectReferenceKind(file, reference.Node, reference.Kind)
		}
	}
	for _, reference := range file.Semantic.UnresolvedReferences() {
		result[reference.Node] = pointerEffectReferenceKind(file, reference.Node, reference.Kind)
	}
	return result
}

func compactEffectReferenceKinds(file *File) map[syntax.NodeID]semantic.ReferenceKind {
	result := make(map[syntax.NodeID]semantic.ReferenceKind)
	for _, symbol := range file.CompactSemantic.Symbols {
		for _, reference := range file.CompactSemantic.References(symbol) {
			result[reference.Node] = compactEffectReferenceKind(file, reference.Node, reference.Kind)
		}
	}
	for _, reference := range file.CompactSemantic.UnresolvedReferences() {
		result[reference.Node] = compactEffectReferenceKind(file, reference.Node, reference.Kind)
	}
	return result
}

func compactEffectReferenceKind(file *File, node syntax.NodeID, kind semantic.ReferenceKind) semantic.ReferenceKind {
	tree := file.CompactWalk
	for current := node; tree.Tree.Valid(current); current = tree.Parent(current) {
		parent := tree.Parent(current)
		if !tree.Tree.Valid(parent) {
			break
		}
		if tree.Tree.Kind(parent) == parser.KindAssignmentExpression && compactEffectInside(tree, current, tree.Tree.Field(parent, "left")) {
			if tree.Tree.TokenKind(parent) == token.Assign {
				return semantic.ReferenceWrite
			}
			return semantic.ReferenceReadWrite
		}
		if tree.Tree.Kind(parent) == parser.KindUpdateExpression {
			return semantic.ReferenceReadWrite
		}
		if tree.Tree.Kind(parent) != parser.KindSubscriptExpression && tree.Tree.Kind(parent) != parser.KindParenthesizedExpression && tree.Tree.Kind(parent) != parser.KindTaggedExpression {
			break
		}
	}
	return kind
}

func compactHasChildToken(tree *walk.CompactModel, node syntax.NodeID, kind token.Kind) bool {
	if tree == nil || !tree.Tree.Valid(node) {
		return false
	}
	for index := 0; index < tree.Tree.ChildCount(node); index++ {
		if tree.Tree.TokenKind(tree.Tree.Child(node, index)) == kind {
			return true
		}
	}
	return false
}

func compactEffectHasDimension(tree *syntax.CompactTree, node syntax.NodeID) bool {
	for index := 0; index < tree.ChildCount(node); index++ {
		if tree.Kind(tree.Child(node, index)) == parser.KindDimension {
			return true
		}
	}
	return false
}

func compactEffectInside(tree *walk.CompactModel, node, container syntax.NodeID) bool {
	return tree.Tree.Valid(node) && tree.Tree.Valid(container) && tree.Tree.Start(node) >= tree.Tree.Start(container) && tree.Tree.End(node) <= tree.Tree.End(container)
}

func pointerEffectReferenceKind(file *File, node *parser.Node, kind semantic.ReferenceKind) semantic.ReferenceKind {
	for current := node; current != nil; current = file.Walk.Parent(current) {
		parent := file.Walk.Parent(current)
		if parent == nil {
			break
		}
		if parent.Kind == parser.KindAssignmentExpression && pointerEffectInside(current, parent.Field("left")) {
			if parent.Tok.Kind == token.Assign {
				return semantic.ReferenceWrite
			}
			return semantic.ReferenceReadWrite
		}
		if parent.Kind == parser.KindUpdateExpression {
			return semantic.ReferenceReadWrite
		}
		if parent.Kind != parser.KindSubscriptExpression && parent.Kind != parser.KindParenthesizedExpression && parent.Kind != parser.KindTaggedExpression {
			break
		}
	}
	return kind
}

func pointerEffectHasDimension(node *parser.Node) bool {
	for _, child := range node.Children {
		if child.Kind == parser.KindDimension {
			return true
		}
	}
	return false
}

func pointerEffectInside(node, container *parser.Node) bool {
	return node != nil && container != nil && node.Start >= container.Start && node.End <= container.End
}

func effectBaseIdentifier(node cst.Node) cst.Node {
	for node.Valid() {
		switch node.Kind() {
		case parser.KindIdentifier:
			return node
		case parser.KindParenthesizedExpression, parser.KindTaggedExpression:
			node = node.Field("expression")
		case parser.KindSubscriptExpression:
			node = node.Field("array")
		default:
			return cst.Node{}
		}
	}
	return cst.Node{}
}

func effectResolvedSymbol(file *File, node cst.Node) (cst.Node, semantic.SymbolKind, bool) {
	if file == nil || !node.Valid() {
		return cst.Node{}, 0, false
	}
	if file.Semantic != nil {
		symbol := file.Semantic.Resolve(node.Pointer())
		if symbol == nil || symbol.Ambiguous {
			return cst.Node{}, 0, false
		}
		return file.Syntax.PointerNode(symbol.Decl), symbol.Kind, true
	}
	symbol := file.CompactSemantic.Resolve(node.ID())
	if symbol == nil || symbol.Ambiguous {
		return cst.Node{}, 0, false
	}
	return file.Syntax.CompactNode(symbol.Decl), symbol.Kind, true
}

func effectReferencesByAmpersand(file *File, parameter cst.Node) bool {
	end := parameter.End()
	if name := parameter.Field("name"); name.Valid() {
		end = name.Start()
	}
	for index := 0; index < file.Syntax.TokenCount(); index++ {
		current := file.Syntax.Token(index)
		if current.Start() >= end {
			break
		}
		if current.Start() >= parameter.Start() && current.End() <= end && current.Kind() == token.Amp {
			return true
		}
	}
	return false
}

func cloneEffectDeclarations(values map[declarationID]Declaration) map[declarationID]Declaration {
	result := make(map[declarationID]Declaration, len(values))
	mergeEffectDeclarations(result, values)
	return result
}

func mergeEffectDeclarations(target, source map[declarationID]Declaration) {
	for key, declaration := range source {
		target[key] = declaration
	}
}

func cloneEffectParameters(values map[int]bool) map[int]bool {
	result := make(map[int]bool, len(values))
	for index := range values {
		result[index] = true
	}
	return result
}

func sameEffectDeclarations(left, right map[declarationID]Declaration) bool {
	if len(left) != len(right) {
		return false
	}
	for key := range left {
		if _, ok := right[key]; !ok {
			return false
		}
	}
	return true
}

func sameEffectParameters(left, right map[int]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !right[index] {
			return false
		}
	}
	return true
}

func publicFunctionEffects(state *functionEffectState) FunctionEffects {
	result := FunctionEffects{Complete: state.complete, Pure: state.pure}
	for _, declaration := range state.reads {
		result.ReadsGlobals = append(result.ReadsGlobals, declaration)
	}
	for _, declaration := range state.writes {
		result.WritesGlobals = append(result.WritesGlobals, declaration)
	}
	for index := range state.mutated {
		result.MutatedParameters = append(result.MutatedParameters, index)
	}
	for _, call := range state.calls {
		result.Calls = append(result.Calls, call.Callee)
	}
	sortDeclarations(result.ReadsGlobals)
	sortDeclarations(result.WritesGlobals)
	sort.Ints(result.MutatedParameters)
	sortDeclarations(result.Calls)
	return result
}
