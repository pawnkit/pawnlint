package project

import (
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
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
	reads               map[string]Declaration
	writes              map[string]Declaration
	mutated             map[int]bool
	calls               []Call
	parameterIndexes    map[*semantic.Symbol]int
	referenceParameters map[*semantic.Symbol]bool
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
	m.effects = make(map[string]FunctionEffects)
	if m.CallGraph == nil {
		return
	}
	states := make(map[string]*functionEffectState, len(m.CallGraph.Functions))
	references := make(map[*File]map[*parser.Node]semantic.ReferenceKind)
	symbols := make(map[*parser.Node][]*semantic.Symbol)
	for _, file := range m.Files {
		references[file] = effectReferenceKinds(file)
		for _, symbol := range file.Semantic.Symbols {
			if symbol.Function != nil {
				symbols[symbol.Function] = append(symbols[symbol.Function], symbol)
			}
		}
	}
	for _, function := range m.CallGraph.Functions {
		states[declarationKey(function)] = m.directFunctionEffects(function, references[function.File], symbols[function.Node])
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

func (m *Model) directFunctionEffects(function Declaration, referenceKinds map[*parser.Node]semantic.ReferenceKind, symbols []*semantic.Symbol) *functionEffectState {
	state := &functionEffectState{
		complete:            true,
		pure:                true,
		reads:               make(map[string]Declaration),
		writes:              make(map[string]Declaration),
		mutated:             make(map[int]bool),
		parameterIndexes:    make(map[*semantic.Symbol]int),
		referenceParameters: make(map[*semantic.Symbol]bool),
	}
	file := function.File
	node := function.Node
	if file == nil || node == nil || node.HasError || file.Walk.Inactive(node) || file.Walk.Uncertain(node) {
		state.complete = false
		state.intrinsicImpure = true
		state.pure = false
		return state
	}
	m.indexEffectParameters(file, node, state)
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
			m.applyDirectReferenceEffect(state, file, current, referenceKinds)
		}
		for _, child := range current.Children {
			visit(child)
		}
	}
	visit(node)
	state.pure = state.complete && !state.intrinsicImpure && len(state.reads) == 0 && len(state.writes) == 0 && len(state.mutated) == 0
	return state
}

func (m *Model) applyDirectReferenceEffect(state *functionEffectState, file *File, identifier *parser.Node, referenceKinds map[*parser.Node]semantic.ReferenceKind) {
	kind, reference := referenceKinds[identifier]
	if !reference {
		return
	}
	symbol := file.Semantic.Resolve(identifier)
	if symbol != nil && symbol.Kind == semantic.SymbolParameter && state.referenceParameters[symbol] && kind != semantic.ReferenceRead {
		state.mutated[state.parameterIndexes[symbol]] = true
	}
	declaration, ok := m.Resolve(file, identifier)
	if !ok || declaration.Kind != semantic.SymbolGlobal {
		return
	}
	if kind == semantic.ReferenceRead && declaration.Symbol != nil && declaration.Symbol.Constant {
		return
	}
	key := declarationKey(declaration)
	if kind == semantic.ReferenceRead {
		state.reads[key] = declaration
	} else {
		state.writes[key] = declaration
	}
}

func (m *Model) indexEffectParameters(file *File, function *parser.Node, state *functionEffectState) {
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
			state.parameterIndexes[symbol] = index
			state.referenceParameters[symbol] = walk.ReferencesByAmpersand(file.Parsed.Tokens, parameter) || effectHasDimension(parameter)
			break
		}
		index++
	}
}

func (m *Model) mergeFunctionEffects(function Declaration, state *functionEffectState, states map[string]*functionEffectState) bool {
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

func (m *Model) mapMutatedArguments(state *functionEffectState, call Call, calleeMutated map[int]bool, mutated map[int]bool, writes map[string]Declaration) bool {
	arguments := call.Node.Field("arguments")
	if arguments == nil {
		return false
	}
	complete := true
	for index := range calleeMutated {
		argumentIndex := index + call.ArgumentOffset
		if argumentIndex < 0 || argumentIndex >= len(arguments.Children) {
			complete = false
			continue
		}
		identifier := effectBaseIdentifier(call.File, arguments.Children[argumentIndex])
		if identifier == nil {
			complete = false
			continue
		}
		if symbol := call.File.Semantic.Resolve(identifier); symbol != nil {
			if parameterIndex, ok := state.parameterIndexes[symbol]; ok && state.referenceParameters[symbol] {
				mutated[parameterIndex] = true
			}
			continue
		}
		if declaration, ok := m.Resolve(call.File, identifier); ok && declaration.Kind == semantic.SymbolGlobal {
			writes[declarationKey(declaration)] = declaration
			continue
		}
		complete = false
	}
	return complete
}

func effectReferenceKinds(file *File) map[*parser.Node]semantic.ReferenceKind {
	result := make(map[*parser.Node]semantic.ReferenceKind)
	for _, symbol := range file.Semantic.Symbols {
		for _, reference := range file.Semantic.References(symbol) {
			result[reference.Node] = effectReferenceKind(file, reference.Node, reference.Kind)
		}
	}
	for _, reference := range file.Semantic.UnresolvedReferences() {
		result[reference.Node] = effectReferenceKind(file, reference.Node, reference.Kind)
	}
	return result
}

func effectReferenceKind(file *File, node *parser.Node, kind semantic.ReferenceKind) semantic.ReferenceKind {
	for current := node; current != nil; current = file.Walk.Parent(current) {
		parent := file.Walk.Parent(current)
		if parent == nil {
			break
		}
		if parent.Kind == parser.KindAssignmentExpression && effectInside(current, parent.Field("left")) {
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

func effectBaseIdentifier(file *File, node *parser.Node) *parser.Node {
	for node != nil {
		switch node.Kind {
		case parser.KindIdentifier:
			return node
		case parser.KindParenthesizedExpression, parser.KindTaggedExpression:
			node = node.Field("expression")
		case parser.KindSubscriptExpression:
			node = node.Field("array")
		default:
			return nil
		}
	}
	return nil
}

func effectHasDimension(node *parser.Node) bool {
	for _, child := range node.Children {
		if child.Kind == parser.KindDimension {
			return true
		}
	}
	return false
}

func effectInside(node, container *parser.Node) bool {
	return node != nil && container != nil && node.Start >= container.Start && node.End <= container.End
}

func cloneEffectDeclarations(values map[string]Declaration) map[string]Declaration {
	result := make(map[string]Declaration, len(values))
	mergeEffectDeclarations(result, values)
	return result
}

func mergeEffectDeclarations(target, source map[string]Declaration) {
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

func sameEffectDeclarations(left, right map[string]Declaration) bool {
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
