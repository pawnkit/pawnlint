package project

import (
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
)

type Call struct {
	Caller Declaration
	Callee Declaration
	File   *File
	Node   *parser.Node
}

type CallGraph struct {
	Functions []Declaration
	Calls     []Call
	outgoing  map[string][]Call
}

func (m *Model) buildCallGraph() *CallGraph {
	graph := &CallGraph{outgoing: make(map[string][]Call)}
	byNode := make(map[*parser.Node]Declaration)
	for _, declarations := range m.Declarations {
		for _, declaration := range declarations {
			if declaration.Kind != semantic.SymbolFunction || declaration.Node.Kind != parser.KindFunctionDefinition || declaration.Symbol == nil || declaration.Symbol.Ambiguous {
				continue
			}
			graph.Functions = append(graph.Functions, declaration)
			byNode[declaration.Node] = declaration
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
			caller := byNode[file.Walk.EnclosingFunction(call)]
			if caller.Node == nil {
				continue
			}
			resolved, ok := m.Resolve(file, calleeNode)
			if !ok {
				continue
			}
			callee, ok := m.callDefinition(file, resolved)
			if !ok {
				continue
			}
			graph.Calls = append(graph.Calls, Call{Caller: caller, Callee: callee, File: file, Node: call})
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
	for _, call := range graph.Calls {
		key := declarationKey(call.Caller)
		graph.outgoing[key] = append(graph.outgoing[key], call)
	}
	return graph
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

func (g *CallGraph) RecursiveComponents() [][]Declaration {
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
