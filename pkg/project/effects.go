package project

import (
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/cst"
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
	if m == nil {
		return FunctionEffects{}, false
	}
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
	references := make(map[*File]map[cst.Node]semantic.ReferenceKind)
	symbols := make(map[cst.Node][]cst.Node)
	symbolKinds := make(map[*File]map[cst.Node]semantic.SymbolKind)
	for _, file := range m.Files {
		references[file] = effectReferenceKinds(file)
		symbolKinds[file] = make(map[cst.Node]semantic.SymbolKind)
		if file.Semantic != nil {
			for _, symbol := range file.Semantic.Symbols {
				declaration := file.Syntax.PointerNode(symbol.Decl)
				symbolKinds[file][declaration] = symbol.Kind
				if symbol.Function != nil {
					function := file.Syntax.PointerNode(symbol.Function)
					symbols[function] = append(symbols[function], declaration)
				}
			}
			continue
		}
		for _, symbol := range file.CompactSemantic.Symbols {
			declaration := file.Syntax.CompactNode(symbol.Decl)
			symbolKinds[file][declaration] = symbol.Kind
			if symbol.Function != syntax.NoNode {
				function := file.Syntax.CompactNode(symbol.Function)
				symbols[function] = append(symbols[function], declaration)
			}
		}
	}
	for _, function := range m.CallGraph.Functions {
		states[declarationKey(function)] = m.directFunctionEffects(function, references[function.File], symbols[declarationSyntax(function)], symbolKinds[function.File])
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

func (m *Model) directFunctionEffects(function Declaration, referenceKinds map[cst.Node]semantic.ReferenceKind, symbols []cst.Node, symbolKinds map[cst.Node]semantic.SymbolKind) *functionEffectState {
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
	node := declarationSyntax(function)
	if file == nil || !node.Valid() || node.HasError() || file.Syntax.Inactive(node) || file.Syntax.Uncertain(node) {
		state.complete = false
		state.intrinsicImpure = true
		state.pure = false
		return state
	}
	m.indexEffectParameters(file, node, state, symbolKinds)
	for _, symbol := range symbols {
		kind, known := symbolKinds[symbol]
		if !known || kind != semantic.SymbolLocal || !file.Syntax.Parent(symbol).HasChildToken(token.KwStatic) {
			continue
		}
		state.intrinsicImpure = true
	}
	state.calls = append([]Call(nil), m.CallGraph.Outgoing(function)...)
	knownCalls := make(map[cst.Node]bool)
	for _, call := range state.calls {
		knownCalls[callSyntax(call)] = true
	}
	var visit func(cst.Node)
	visit = func(current cst.Node) {
		if !current.Valid() || file.Syntax.Inactive(current) {
			return
		}
		if !current.Same(node) && current.Kind() == parser.KindFunctionDefinition {
			state.complete = false
			return
		}
		if file.Syntax.Uncertain(current) {
			state.complete = false
			return
		}
		switch current.Kind() {
		case parser.KindMacroInvocation, parser.KindMacroInvocationBlock, parser.KindConditionalSplice:
			state.complete = false
		case parser.KindStateStatement:
			state.intrinsicImpure = true
		case parser.KindCallExpression:
			if !knownCalls[current] {
				state.complete = false
			}
		case parser.KindIdentifier:
			m.applyDirectReferenceEffect(state, file, current, referenceKinds)
		}
		for index := 0; index < current.ChildCount(); index++ {
			visit(current.Child(index))
		}
	}
	visit(node)
	state.pure = state.complete && !state.intrinsicImpure && len(state.reads) == 0 && len(state.writes) == 0 && len(state.mutated) == 0
	return state
}

func (m *Model) applyDirectReferenceEffect(state *functionEffectState, file *File, identifier cst.Node, referenceKinds map[cst.Node]semantic.ReferenceKind) {
	kind, reference := referenceKinds[identifier]
	if !reference {
		return
	}
	symbol, symbolKind, symbolKnown := effectResolvedSymbol(file, identifier)
	if symbol.Valid() && symbolKnown && symbolKind == semantic.SymbolParameter && state.referenceParameters[symbol] && kind != semantic.ReferenceRead {
		state.mutated[state.parameterIndexes[symbol]] = true
	}
	declaration, ok := m.resolveSyntax(file, identifier)
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

func (m *Model) indexEffectParameters(file *File, function cst.Node, state *functionEffectState, symbolKinds map[cst.Node]semantic.SymbolKind) {
	list := function.Field("parameters")
	if !list.Valid() {
		return
	}
	index := 0
	for child := 0; child < list.ChildCount(); child++ {
		parameter := list.Child(child)
		if parameter.Kind() != parser.KindParameter {
			continue
		}
		symbol := parameter
		symbolKind, symbolKnown := symbolKinds[symbol]
		if symbol.Valid() && symbolKnown && symbolKind == semantic.SymbolParameter {
			state.parameterIndexes[symbol] = index
			state.referenceParameters[symbol] = effectReferencesByAmpersand(file, parameter) || effectHasDimension(parameter)
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

func effectReferenceKinds(file *File) map[cst.Node]semantic.ReferenceKind {
	result := make(map[cst.Node]semantic.ReferenceKind)
	if file.Semantic != nil {
		for _, symbol := range file.Semantic.Symbols {
			for _, reference := range file.Semantic.References(symbol) {
				node := file.Syntax.PointerNode(reference.Node)
				result[node] = effectReferenceKind(file, node, reference.Kind)
			}
		}
		for _, reference := range file.Semantic.UnresolvedReferences() {
			node := file.Syntax.PointerNode(reference.Node)
			result[node] = effectReferenceKind(file, node, reference.Kind)
		}
		return result
	}
	for _, symbol := range file.CompactSemantic.Symbols {
		for _, reference := range file.CompactSemantic.References(symbol) {
			node := file.Syntax.CompactNode(reference.Node)
			result[node] = effectReferenceKind(file, node, reference.Kind)
		}
	}
	for _, reference := range file.CompactSemantic.UnresolvedReferences() {
		node := file.Syntax.CompactNode(reference.Node)
		result[node] = effectReferenceKind(file, node, reference.Kind)
	}
	return result
}

func effectReferenceKind(file *File, node cst.Node, kind semantic.ReferenceKind) semantic.ReferenceKind {
	for current := node; current.Valid(); current = file.Syntax.Parent(current) {
		parent := file.Syntax.Parent(current)
		if !parent.Valid() {
			break
		}
		if parent.Kind() == parser.KindAssignmentExpression && effectInside(current, parent.Field("left")) {
			if parent.TokenKind() == token.Assign {
				return semantic.ReferenceWrite
			}
			return semantic.ReferenceReadWrite
		}
		if parent.Kind() == parser.KindUpdateExpression {
			return semantic.ReferenceReadWrite
		}
		if parent.Kind() != parser.KindSubscriptExpression && parent.Kind() != parser.KindParenthesizedExpression && parent.Kind() != parser.KindTaggedExpression {
			break
		}
	}
	return kind
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

func effectHasDimension(node cst.Node) bool {
	for index := 0; index < node.ChildCount(); index++ {
		if node.Child(index).Kind() == parser.KindDimension {
			return true
		}
	}
	return false
}

func effectInside(node, container cst.Node) bool {
	return node.Valid() && container.Valid() && node.Start() >= container.Start() && node.End() <= container.End()
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
