package project

import (
	"sort"
	"strconv"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
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
	outgoing      map[string][]Call
	asyncOutgoing map[string][]Call
	asyncIncoming map[string][]Call
	recursive     [][]Declaration
}

func (m *Model) buildCallGraph() *CallGraph {
	graph := &CallGraph{outgoing: make(map[string][]Call), asyncOutgoing: make(map[string][]Call), asyncIncoming: make(map[string][]Call)}
	byNode := make(map[*File]map[*parser.Node]Declaration)
	for _, declarations := range m.Declarations {
		for _, declaration := range declarations {
			if declaration.Kind != semantic.SymbolFunction || declaration.Node.Kind != parser.KindFunctionDefinition || declaration.Symbol == nil || declaration.Symbol.Ambiguous {
				continue
			}
			graph.Functions = append(graph.Functions, declaration)
			if byNode[declaration.File] == nil {
				byNode[declaration.File] = make(map[*parser.Node]Declaration)
			}
			byNode[declaration.File][declaration.Node] = declaration
		}
	}
	sortDeclarations(graph.Functions)
	for _, file := range m.Files {
		for _, call := range file.Walk.OfKind(parser.KindCallExpression) {
			if file.Walk.Uncertain(call) || file.Walk.Inactive(call) {
				continue
			}
			calleeNode := call.Field("function")
			if calleeNode == nil || calleeNode.Kind != parser.KindIdentifier {
				continue
			}
			caller := byNode[file][file.Walk.EnclosingFunction(call)]
			if caller.Node == nil {
				continue
			}
			callees := m.FunctionVariants(file, calleeNode)
			if len(callees) == 0 {
				resolved, ok := m.Resolve(file, calleeNode)
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
				graph.Calls = append(graph.Calls, Call{Caller: caller, Callee: callee, File: file, Node: call})
			}
		}
	}
	sort.SliceStable(graph.Calls, func(i, j int) bool {
		left, right := graph.Calls[i], graph.Calls[j]
		if declarationKey(left.Caller) != declarationKey(right.Caller) {
			return declarationKey(left.Caller) < declarationKey(right.Caller)
		}
		if left.File.canonical != right.File.canonical {
			return left.File.canonical < right.File.canonical
		}
		if left.Node.Start != right.Node.Start {
			return left.Node.Start < right.Node.Start
		}
		return declarationKey(left.Callee) < declarationKey(right.Callee)
	})
	graph.buildRuntimeEdges(m, byNode)
	graph.recursive = graph.findRecursiveComponents()
	return graph
}

func (g *CallGraph) buildRuntimeEdges(model *Model, byNode map[*File]map[*parser.Node]Declaration) {
	for _, function := range g.Functions {
		if function.Name == "main" {
			g.EntryPoints = append(g.EntryPoints, EntryPoint{Function: function, Kind: EntryMain})
		} else if walk.HasChildToken(function.Node, token.KwPublic) {
			g.EntryPoints = append(g.EntryPoints, EntryPoint{Function: function, Kind: EntryCallback})
		}
	}
	for _, file := range model.Files {
		tree, parsed := file.ExpandedWalk, file.ExpandedParsed
		if !file.ExpansionComplete || tree == nil || parsed == nil {
			continue
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
			caller := runtimeCaller(file, tree.EnclosingFunction(call), byNode[file], model.Declarations)
			if caller.Node == nil {
				continue
			}
			argumentOffset := -1
			if name == "SetTimerEx" {
				argumentOffset = 4
			} else if name == "CallLocalFunction" || name == "CallRemoteFunction" {
				argumentOffset = 2
			}
			for _, targetFunction := range model.runtimeDefinitions(file, target) {
				resolved := Call{Caller: caller, Callee: targetFunction, File: file, Node: call, ArgumentOffset: argumentOffset}
				if name == "CallLocalFunction" || name == "CallRemoteFunction" {
					resolved.Kind = CallDynamic
					g.Calls = append(g.Calls, resolved)
				} else {
					resolved.Kind = CallTimer
					g.AsyncCalls = append(g.AsyncCalls, resolved)
				}
			}
		}
	}
	sort.SliceStable(g.EntryPoints, func(i, j int) bool {
		return declarationKey(g.EntryPoints[i].Function) < declarationKey(g.EntryPoints[j].Function)
	})
	sort.SliceStable(g.AsyncCalls, func(i, j int) bool {
		left, right := g.AsyncCalls[i], g.AsyncCalls[j]
		if declarationKey(left.Caller) != declarationKey(right.Caller) {
			return declarationKey(left.Caller) < declarationKey(right.Caller)
		}
		if left.Node.Start != right.Node.Start {
			return left.Node.Start < right.Node.Start
		}
		return declarationKey(left.Callee) < declarationKey(right.Callee)
	})
	sort.SliceStable(g.Calls, func(i, j int) bool {
		left, right := g.Calls[i], g.Calls[j]
		if declarationKey(left.Caller) != declarationKey(right.Caller) {
			return declarationKey(left.Caller) < declarationKey(right.Caller)
		}
		if left.File.canonical != right.File.canonical {
			return left.File.canonical < right.File.canonical
		}
		if left.Node.Start != right.Node.Start {
			return left.Node.Start < right.Node.Start
		}
		return declarationKey(left.Callee) < declarationKey(right.Callee)
	})
	g.outgoing = make(map[string][]Call)
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

func runtimeCaller(file *File, function *parser.Node, direct map[*parser.Node]Declaration, declarations map[string][]Declaration) Declaration {
	if caller := direct[function]; caller.Node != nil {
		return caller
	}
	if function == nil || file.ExpandedWalk == nil {
		return Declaration{}
	}
	name := file.ExpandedWalk.Text(function.Field("name"))
	var result Declaration
	for _, declaration := range declarations[name] {
		if declaration.File != file || declaration.Node.Kind != parser.KindFunctionDefinition || declaration.Symbol == nil || declaration.Symbol.Ambiguous {
			continue
		}
		if result.Node != nil {
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
	seen := make(map[string]Declaration)
	for _, unit := range m.Units {
		if _, included := unit.members[file]; !included {
			continue
		}
		for _, declaration := range m.Declarations[name] {
			if _, included := unit.members[declaration.File]; included && declaration.Node.Kind == parser.KindFunctionDefinition && declaration.Symbol != nil && !declaration.Symbol.Ambiguous {
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
			if !projectStateVariantsCoexist(result[left].Symbol, result[right].Symbol) {
				return nil
			}
		}
	}
	return result
}

func (m *Model) callDefinition(from *File, resolved Declaration) (Declaration, bool) {
	if resolved.Node != nil && resolved.Node.Kind == parser.KindFunctionDefinition && resolved.Symbol != nil && !resolved.Symbol.Ambiguous {
		return resolved, true
	}
	seen := make(map[string]Declaration)
	for _, unit := range m.Units {
		contains := false
		for _, file := range unit.Files {
			contains = contains || file == from
		}
		if !contains {
			continue
		}
		for _, declaration := range m.Declarations[resolved.Name] {
			if declaration.Node == nil || declaration.Node.Kind != parser.KindFunctionDefinition || declaration.Symbol == nil || declaration.Symbol.Ambiguous {
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
	indices := make(map[string]int)
	lowlink := make(map[string]int)
	onStack := make(map[string]bool)
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
		return declarationKey(components[i][0]) < declarationKey(components[j][0])
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
