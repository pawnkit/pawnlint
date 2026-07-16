package project

import (
	"sort"
	"strconv"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/cst"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

type CallKind uint8

const (
	CallDirect CallKind = iota
	CallDynamic
	CallTimer
)

type EntryKind uint8

const (
	EntryCallback EntryKind = iota
	EntryMain
)

type Call struct {
	Caller         Declaration
	Callee         Declaration
	File           *File
	Node           *parser.Node
	Kind           CallKind
	ArgumentOffset int
	syntax         cst.Node
}

type EntryPoint struct {
	Function Declaration
	Kind     EntryKind
}

type CallGraph struct {
	Functions     []Declaration
	Calls         []Call
	AsyncCalls    []Call
	EntryPoints   []EntryPoint
	outgoing      map[declarationID][]Call
	asyncOutgoing map[declarationID][]Call
	asyncIncoming map[declarationID][]Call
	recursive     [][]Declaration
}

type runtimeCallFact struct {
	caller         string
	target         string
	node           *parser.Node
	kind           CallKind
	argumentOffset int
	syntax         cst.Node
}

type expansionOriginFact struct {
	span  token.Span
	macro string
}

func (m *Model) buildCallGraph() *CallGraph {
	graph := &CallGraph{outgoing: make(map[declarationID][]Call), asyncOutgoing: make(map[declarationID][]Call), asyncIncoming: make(map[declarationID][]Call)}
	byNode := make(map[*File]map[cst.Node]Declaration)
	for _, declarations := range m.Declarations {
		for _, declaration := range declarations {
			if declaration.Kind != semantic.SymbolFunction || declarationSyntax(declaration).Kind() != parser.KindFunctionDefinition || declarationSymbolAmbiguous(declaration) {
				continue
			}
			graph.Functions = append(graph.Functions, declaration)
			if byNode[declaration.File] == nil {
				byNode[declaration.File] = make(map[cst.Node]Declaration)
			}
			byNode[declaration.File][declarationSyntax(declaration)] = declaration
		}
	}
	sortDeclarations(graph.Functions)
	for _, file := range m.Files {
		for _, call := range file.Syntax.OfKind(parser.KindCallExpression) {
			if file.Syntax.Uncertain(call) || file.Syntax.Inactive(call) {
				continue
			}
			calleeNode := call.Field("function")
			if !calleeNode.Valid() || calleeNode.Kind() != parser.KindIdentifier {
				continue
			}
			caller := byNode[file][file.Syntax.EnclosingFunction(call)]
			if !declarationSyntax(caller).Valid() {
				continue
			}
			callees := m.functionVariants(file, calleeNode)
			if len(callees) == 0 {
				resolved, ok := m.resolveSyntax(file, calleeNode)
				if !ok {
					continue
				}
				callee, ok := m.callDefinition(file, resolved)
				if !ok {
					continue
				}
				callees = []Declaration{callee}
			}
			for _, callee := range callees {
				graph.Calls = append(graph.Calls, Call{Caller: caller, Callee: callee, File: file, Node: call.Pointer(), syntax: call})
			}
		}
	}
	sort.SliceStable(graph.Calls, func(i, j int) bool {
		left, right := graph.Calls[i], graph.Calls[j]
		if declarationKey(left.Caller) != declarationKey(right.Caller) {
			return declarationLess(left.Caller, right.Caller)
		}
		if left.File.canonical != right.File.canonical {
			return left.File.canonical < right.File.canonical
		}
		if callSyntaxOffset(left) != callSyntaxOffset(right) {
			return callSyntaxOffset(left) < callSyntaxOffset(right)
		}
		return declarationLess(left.Callee, right.Callee)
	})
	graph.buildRuntimeEdges(m)
	graph.recursive = graph.findRecursiveComponents()
	return graph
}

func (m *Model) captureRuntimeCalls(file *File) {
	tree, parsed := file.ExpandedWalk, file.ExpandedParsed
	if !file.ExpansionComplete || tree == nil || parsed == nil {
		return
	}
	for _, call := range tree.OfKind(parser.KindCallExpression) {
		if tree.Inactive(call) || tree.Uncertain(call) {
			continue
		}
		callee := call.Field("function")
		if callee == nil || callee.Kind != parser.KindIdentifier {
			continue
		}
		name := tree.Text(callee)
		if name != "SetTimer" && name != "SetTimerEx" && name != "__settimer" && name != "CallLocalFunction" && name != "CallRemoteFunction" {
			continue
		}
		arguments := call.Field("arguments")
		if arguments == nil || len(arguments.Children) == 0 {
			continue
		}
		target, ok := runtimeCallbackName(tree, parsed.Source, arguments.Children[0])
		if !ok {
			continue
		}
		function := tree.EnclosingFunction(call)
		if function == nil {
			continue
		}
		caller := tree.Text(function.Field("name"))
		if caller == "" {
			continue
		}
		node := compactRuntimeCallNode(file, parsed, call, callee)
		fact := runtimeCallFact{caller: caller, target: target, node: node, kind: CallTimer, argumentOffset: -1}
		if name == "SetTimerEx" {
			fact.argumentOffset = 4
		} else if name == "CallLocalFunction" || name == "CallRemoteFunction" {
			fact.kind = CallDynamic
			fact.argumentOffset = 2
		}
		file.runtimeCalls = append(file.runtimeCalls, fact)
		file.captureExpansionOrigins(parsed, call, node)
	}
}

func (m *Model) captureCompactRuntimeCalls(file *File, parsed *parser.CompactFile, tree *walk.CompactModel) {
	if !file.ExpansionComplete || tree == nil || parsed == nil {
		return
	}
	for _, call := range tree.OfKind(parser.KindCallExpression) {
		if tree.Inactive(call) || tree.Uncertain(call) {
			continue
		}
		callee := tree.Tree.Field(call, "function")
		if callee == syntax.NoNode || tree.Tree.Kind(callee) != parser.KindIdentifier {
			continue
		}
		name := tree.Text(callee)
		if name != "SetTimer" && name != "SetTimerEx" && name != "__settimer" && name != "CallLocalFunction" && name != "CallRemoteFunction" {
			continue
		}
		arguments := tree.Tree.Field(call, "arguments")
		if arguments == syntax.NoNode || tree.Tree.ChildCount(arguments) == 0 {
			continue
		}
		target, ok := compactRuntimeCallbackName(tree, tree.Tree.Child(arguments, 0))
		if !ok {
			continue
		}
		function := tree.EnclosingFunction(call)
		if function == syntax.NoNode {
			continue
		}
		caller := tree.Text(tree.Tree.Field(function, "name"))
		if caller == "" {
			continue
		}
		node, physical := compactRuntimeCallNodeFromTree(file, parsed, tree, call, callee)
		fact := runtimeCallFact{caller: caller, target: target, node: node, kind: CallTimer, argumentOffset: -1, syntax: physical}
		if name == "SetTimerEx" {
			fact.argumentOffset = 4
		} else if name == "CallLocalFunction" || name == "CallRemoteFunction" {
			fact.kind = CallDynamic
			fact.argumentOffset = 2
		}
		file.runtimeCalls = append(file.runtimeCalls, fact)
		file.captureCompactExpansionOrigins(parsed, tree, call, node)
	}
}

func compactRuntimeCallbackName(tree *walk.CompactModel, node syntax.NodeID) (string, bool) {
	for tree.Tree.Valid(node) && tree.Tree.Kind(node) == parser.KindParenthesizedExpression {
		node = tree.Tree.Field(node, "expression")
	}
	if !tree.Tree.Valid(node) || tree.Tree.HasError(node) {
		return "", false
	}
	if tree.Tree.Kind(node) == parser.KindStringConcat {
		var result strings.Builder
		for index := 0; index < tree.Tree.ChildCount(node); index++ {
			part, ok := compactRuntimeCallbackName(tree, tree.Tree.Child(node, index))
			if !ok {
				return "", false
			}
			result.WriteString(part)
		}
		return result.String(), true
	}
	if tree.Tree.Kind(node) != parser.KindLiteral || tree.Tree.TokenKind(node) != token.StringLiteral && tree.Tree.TokenKind(node) != token.PackedString {
		return "", false
	}
	raw := strings.TrimPrefix(tree.Tree.TokenText(node), "!")
	value, err := strconv.Unquote(raw)
	if err != nil || value == "" {
		return "", false
	}
	return value, true
}

func compactRuntimeCallNodeFromTree(file *File, parsed *parser.CompactFile, tree *walk.CompactModel, call, callee syntax.NodeID) (*parser.Node, cst.Node) {
	origin := compactNodeOrigin(parsed, tree, callee)
	var location *parser.CompactOrigin
	for origin != 0 && int(origin) < len(parsed.Origins) {
		current := &parsed.Origins[origin]
		if current.File == file.sourceID {
			location = current
			for _, original := range file.Syntax.OfKind(parser.KindCallExpression) {
				function := original.Field("function")
				if function.Valid() && function.Start() == int(current.Start.Offset) {
					return original.Pointer(), original
				}
			}
		}
		origin = current.Parent
	}
	if location == nil {
		return &parser.Node{Kind: parser.KindCallExpression, Start: tree.Tree.Start(call), End: tree.Tree.End(call)}, cst.Node{}
	}
	return &parser.Node{Kind: parser.KindCallExpression, Start: int(location.Start.Offset), End: int(location.End.Offset)}, cst.Node{}
}

func compactNodeOrigin(parsed *parser.CompactFile, tree *walk.CompactModel, node syntax.NodeID) uint32 {
	start, end := tree.Tree.TokenStart(node), tree.Tree.TokenEnd(node)
	for _, current := range parsed.Tokens {
		if int(current.Start.Offset) == start && int(current.End.Offset) == end && current.Kind == tree.Tree.TokenKind(node) {
			return current.Origin
		}
	}
	return 0
}

func (f *File) captureCompactExpansionOrigins(parsed *parser.CompactFile, tree *walk.CompactModel, expanded syntax.NodeID, compact *parser.Node) {
	start, end := tree.Tree.Start(expanded), tree.Tree.End(expanded)
	for _, current := range parsed.Tokens {
		if current.Kind == token.EOF || int(current.End.Offset) <= start || int(current.Start.Offset) >= end || current.Origin == 0 {
			continue
		}
		if f.expansionOrigins == nil {
			f.expansionOrigins = make(map[*parser.Node][]expansionOriginFact)
		}
		for origin := current.Origin; origin != 0 && int(origin) < len(parsed.Origins); origin = parsed.Origins[origin].Parent {
			value := parsed.Origins[origin]
			macro := ""
			if int(value.Macro) < len(parsed.MacroNames) {
				macro = parsed.MacroNames[value.Macro]
			}
			span := token.Span{
				File:  value.File,
				Start: token.Position{Offset: int(value.Start.Offset), Line: int(value.Start.Line), Col: int(value.Start.Col)},
				End:   token.Position{Offset: int(value.End.Offset), Line: int(value.End.Line), Col: int(value.End.Col)},
			}
			f.expansionOrigins[compact] = append(f.expansionOrigins[compact], expansionOriginFact{span: span, macro: macro})
		}
		return
	}
}

func compactRuntimeCallNode(file *File, parsed *parser.File, call, callee *parser.Node) *parser.Node {
	if parsed == file.Parsed {
		return call
	}
	var location *token.Origin
	for origin := callee.Tok.Origin; origin != nil; origin = origin.Parent {
		if origin.Span.File != file.sourceID {
			continue
		}
		location = origin
		for _, original := range file.Walk.OfKind(parser.KindCallExpression) {
			function := original.Field("function")
			if function != nil && function.Start == origin.Span.Start.Offset {
				return original
			}
		}
	}
	if location == nil {
		return &parser.Node{Kind: parser.KindCallExpression, Start: call.Start, End: call.End}
	}
	return &parser.Node{Kind: parser.KindCallExpression, Start: location.Span.Start.Offset, End: location.Span.End.Offset}
}

func (f *File) captureExpansionOrigins(parsed *parser.File, expanded, compact *parser.Node) {
	for _, current := range parsed.Tokens {
		if current.Kind == token.EOF || current.End.Offset <= expanded.Start || current.Start.Offset >= expanded.End || current.Origin == nil {
			continue
		}
		if f.expansionOrigins == nil {
			f.expansionOrigins = make(map[*parser.Node][]expansionOriginFact)
		}
		for origin := current.Origin; origin != nil; origin = origin.Parent {
			f.expansionOrigins[compact] = append(f.expansionOrigins[compact], expansionOriginFact{span: origin.Span, macro: origin.Macro})
		}
		return
	}
}

func (g *CallGraph) buildRuntimeEdges(model *Model) {
	for _, function := range g.Functions {
		if function.Name == "main" {
			g.EntryPoints = append(g.EntryPoints, EntryPoint{Function: function, Kind: EntryMain})
		} else if declarationSyntax(function).HasChildToken(token.KwPublic) {
			g.EntryPoints = append(g.EntryPoints, EntryPoint{Function: function, Kind: EntryCallback})
		}
	}
	for _, file := range model.Files {
		for _, fact := range file.runtimeCalls {
			caller := runtimeCaller(file, fact.caller, model.Declarations)
			if !declarationSyntax(caller).Valid() {
				continue
			}
			for _, targetFunction := range model.runtimeDefinitions(file, fact.target) {
				resolved := Call{Caller: caller, Callee: targetFunction, File: file, Node: fact.node, Kind: fact.kind, ArgumentOffset: fact.argumentOffset, syntax: fact.syntax}
				if fact.kind == CallDynamic {
					g.Calls = append(g.Calls, resolved)
				} else {
					g.AsyncCalls = append(g.AsyncCalls, resolved)
				}
			}
		}
	}
	sort.SliceStable(g.EntryPoints, func(i, j int) bool {
		return declarationLess(g.EntryPoints[i].Function, g.EntryPoints[j].Function)
	})
	sort.SliceStable(g.AsyncCalls, func(i, j int) bool {
		left, right := g.AsyncCalls[i], g.AsyncCalls[j]
		if declarationKey(left.Caller) != declarationKey(right.Caller) {
			return declarationLess(left.Caller, right.Caller)
		}
		if callSyntaxOffset(left) != callSyntaxOffset(right) {
			return callSyntaxOffset(left) < callSyntaxOffset(right)
		}
		return declarationLess(left.Callee, right.Callee)
	})
	sort.SliceStable(g.Calls, func(i, j int) bool {
		left, right := g.Calls[i], g.Calls[j]
		if declarationKey(left.Caller) != declarationKey(right.Caller) {
			return declarationLess(left.Caller, right.Caller)
		}
		if left.File.canonical != right.File.canonical {
			return left.File.canonical < right.File.canonical
		}
		if callSyntaxOffset(left) != callSyntaxOffset(right) {
			return callSyntaxOffset(left) < callSyntaxOffset(right)
		}
		return declarationLess(left.Callee, right.Callee)
	})
	g.outgoing = make(map[declarationID][]Call)
	for _, call := range g.Calls {
		key := declarationKey(call.Caller)
		g.outgoing[key] = append(g.outgoing[key], call)
	}
	for _, call := range g.AsyncCalls {
		key := declarationKey(call.Caller)
		g.asyncOutgoing[key] = append(g.asyncOutgoing[key], call)
		calleeKey := declarationKey(call.Callee)
		g.asyncIncoming[calleeKey] = append(g.asyncIncoming[calleeKey], call)
	}
}

func runtimeCaller(file *File, name string, declarations map[string][]Declaration) Declaration {
	var result Declaration
	for _, declaration := range declarations[name] {
		if declaration.File != file || declarationSyntax(declaration).Kind() != parser.KindFunctionDefinition || declarationSymbolAmbiguous(declaration) {
			continue
		}
		if declarationSyntax(result).Valid() {
			return Declaration{}
		}
		result = declaration
	}
	return result
}

func runtimeCallbackName(tree *walk.Model, source []byte, node *parser.Node) (string, bool) {
	for node != nil && node.Kind == parser.KindParenthesizedExpression {
		node = node.Field("expression")
	}
	if node == nil || node.HasError {
		return "", false
	}
	if node.Kind == parser.KindStringConcat {
		var result strings.Builder
		for _, child := range node.Children {
			part, ok := runtimeCallbackName(tree, source, child)
			if !ok {
				return "", false
			}
			result.WriteString(part)
		}
		return result.String(), true
	}
	if node.Kind != parser.KindLiteral || node.Tok.Kind != token.StringLiteral && node.Tok.Kind != token.PackedString {
		return "", false
	}
	raw := node.Tok.Text(source)
	raw = strings.TrimPrefix(raw, "!")
	value, err := strconv.Unquote(raw)
	if err != nil || value == "" {
		return "", false
	}
	return value, true
}

func (m *Model) runtimeDefinitions(file *File, name string) []Declaration {
	seen := make(map[declarationID]Declaration)
	for _, unit := range m.Units {
		if _, included := unit.members[file]; !included {
			continue
		}
		for _, declaration := range m.Declarations[name] {
			if _, included := unit.members[declaration.File]; included && declarationSyntax(declaration).Kind() == parser.KindFunctionDefinition && !declarationSymbolAmbiguous(declaration) {
				seen[declarationKey(declaration)] = declaration
			}
		}
	}
	result := make([]Declaration, 0, len(seen))
	for _, declaration := range seen {
		result = append(result, declaration)
	}
	sortDeclarations(result)
	for left := range result {
		for right := left + 1; right < len(result); right++ {
			if !projectStateVariantsCoexist(result[left], result[right]) {
				return nil
			}
		}
	}
	return result
}

func (m *Model) callDefinition(from *File, resolved Declaration) (Declaration, bool) {
	if declarationSyntax(resolved).Kind() == parser.KindFunctionDefinition && !declarationSymbolAmbiguous(resolved) {
		return resolved, true
	}
	seen := make(map[declarationID]Declaration)
	for _, unit := range m.Units {
		contains := false
		for _, file := range unit.Files {
			contains = contains || file == from
		}
		if !contains {
			continue
		}
		for _, declaration := range m.Declarations[resolved.Name] {
			if declarationSyntax(declaration).Kind() != parser.KindFunctionDefinition || declarationSymbolAmbiguous(declaration) {
				continue
			}
			for _, file := range unit.Files {
				if declaration.File == file {
					seen[declarationKey(declaration)] = declaration
				}
			}
		}
	}
	if len(seen) != 1 {
		return Declaration{}, false
	}
	for _, declaration := range seen {
		return declaration, true
	}
	return Declaration{}, false
}

func callSyntax(call Call) cst.Node {
	if call.syntax.Valid() {
		return call.syntax
	}
	if call.File != nil && call.File.Syntax != nil {
		return call.File.Syntax.PointerNode(call.Node)
	}
	return cst.Node{}
}

func callSyntaxOffset(call Call) int {
	return callSyntax(call).Start()
}

func (g *CallGraph) Outgoing(function Declaration) []Call {
	if g == nil {
		return nil
	}
	return append([]Call(nil), g.outgoing[declarationKey(function)]...)
}

func (g *CallGraph) AsyncOutgoing(function Declaration) []Call {
	if g == nil {
		return nil
	}
	return append([]Call(nil), g.asyncOutgoing[declarationKey(function)]...)
}

func (g *CallGraph) AsyncIncoming(function Declaration) []Call {
	if g == nil {
		return nil
	}
	return append([]Call(nil), g.asyncIncoming[declarationKey(function)]...)
}

func (g *CallGraph) RecursiveComponents() [][]Declaration {
	if g == nil {
		return nil
	}
	result := make([][]Declaration, len(g.recursive))
	for index, component := range g.recursive {
		result[index] = append([]Declaration(nil), component...)
	}
	return result
}

func (g *CallGraph) findRecursiveComponents() [][]Declaration {
	if g == nil {
		return nil
	}
	index := 0
	indices := make(map[declarationID]int)
	lowlink := make(map[declarationID]int)
	onStack := make(map[declarationID]bool)
	stack := make([]Declaration, 0, len(g.Functions))
	var components [][]Declaration
	var connect func(Declaration)
	connect = func(function Declaration) {
		key := declarationKey(function)
		index++
		indices[key] = index
		lowlink[key] = index
		stack = append(stack, function)
		onStack[key] = true
		for _, call := range g.outgoing[key] {
			calleeKey := declarationKey(call.Callee)
			if indices[calleeKey] == 0 {
				connect(call.Callee)
				if lowlink[calleeKey] < lowlink[key] {
					lowlink[key] = lowlink[calleeKey]
				}
			} else if onStack[calleeKey] && indices[calleeKey] < lowlink[key] {
				lowlink[key] = indices[calleeKey]
			}
		}
		if lowlink[key] != indices[key] {
			return
		}
		var component []Declaration
		for len(stack) != 0 {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			lastKey := declarationKey(last)
			onStack[lastKey] = false
			component = append(component, last)
			if lastKey == key {
				break
			}
		}
		if len(component) == 1 && !g.selfCalls(component[0]) {
			return
		}
		sortDeclarations(component)
		components = append(components, component)
	}
	for _, function := range g.Functions {
		if indices[declarationKey(function)] == 0 {
			connect(function)
		}
	}
	sort.SliceStable(components, func(i, j int) bool {
		return declarationLess(components[i][0], components[j][0])
	})
	return components
}

func (g *CallGraph) selfCalls(function Declaration) bool {
	key := declarationKey(function)
	for _, call := range g.outgoing[key] {
		if declarationKey(call.Callee) == key {
			return true
		}
	}
	return false
}
