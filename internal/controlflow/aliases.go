package controlflow

import (
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

type aliasState map[*semantic.Symbol]int

type aliasEvent struct {
	offset int
	target *semantic.Symbol
	source *semantic.Symbol
}

func (m *Model) Aliases(node *parser.Node, symbol *semantic.Symbol) []*semantic.Symbol {
	if m == nil || node == nil || symbol == nil {
		return nil
	}
	for _, function := range m.Functions {
		block := function.Block(node)
		if block == nil || function.Uncertain || !function.ReachableBlock(block) || function.aliasIndexes[symbol] == 0 {
			continue
		}
		state := copyAliasState(function.aliasIn[block])
		applyAliasEvents(state, function.aliasEvents[block], node.Start, function.aliasIndexes)
		class := state[symbol]
		var result []*semantic.Symbol
		for candidate, candidateClass := range state {
			if candidateClass == class {
				result = append(result, candidate)
			}
		}
		sort.SliceStable(result, func(i, j int) bool {
			if result[i].Name != result[j].Name {
				return result[i].Name < result[j].Name
			}
			return result[i].NameNode.Start < result[j].NameNode.Start
		})
		return result
	}
	return nil
}

func buildAliasFlow(function *Function, tree *walk.Model, semantics *semantic.Model, options Options) {
	function.aliasIn = make(map[*Block]aliasState)
	function.aliasEvents = make(map[*Block][]aliasEvent)
	function.aliasIndexes = make(map[*semantic.Symbol]int)
	if function.Uncertain || semantics == nil {
		return
	}
	eligible := make(map[*semantic.Symbol]bool)
	for _, symbol := range semantics.Symbols {
		if aliasSymbol(tree, function, symbol) {
			eligible[symbol] = true
			function.aliasIndexes[symbol] = len(function.aliasIndexes) + 1
		}
	}
	for symbol := range eligible {
		if symbol.Kind != semantic.SymbolLocal {
			continue
		}
		block := function.Block(symbol.Decl)
		if block == nil {
			continue
		}
		function.aliasEvents[block] = append(function.aliasEvents[block], aliasEvent{offset: symbol.Decl.End, target: symbol, source: aliasExpressionSymbol(symbol.Decl.Field("initializer"), semantics, eligible)})
	}
	for _, node := range tree.OfKind(parser.KindAssignmentExpression) {
		if tree.EnclosingFunction(node) != function.Node {
			continue
		}
		target := aliasExpressionSymbol(node.Field("left"), semantics, eligible)
		block := function.Block(node)
		if target == nil || block == nil {
			continue
		}
		var source *semantic.Symbol
		if node.Tok.Kind == token.Assign && !conditionallyExecuted(tree, node) {
			source = aliasExpressionSymbol(node.Field("right"), semantics, eligible)
		}
		function.aliasEvents[block] = append(function.aliasEvents[block], aliasEvent{offset: node.End, target: target, source: source})
	}
	for _, node := range tree.OfKind(parser.KindUpdateExpression) {
		if tree.EnclosingFunction(node) != function.Node {
			continue
		}
		target := aliasExpressionSymbol(node.Field("expression"), semantics, eligible)
		block := function.Block(node)
		if target != nil && block != nil {
			function.aliasEvents[block] = append(function.aliasEvents[block], aliasEvent{offset: node.End, target: target})
		}
	}
	for _, node := range tree.OfKind(parser.KindCallExpression) {
		if tree.EnclosingFunction(node) != function.Node {
			continue
		}
		block := function.Block(node)
		arguments := node.Field("arguments")
		if block == nil || arguments == nil {
			continue
		}
		effects, known := callEffects(node, semantics, options)
		indexes := effects.MutatedArguments
		if !known || !effects.Complete {
			indexes = make([]int, len(arguments.Children))
			for index := range arguments.Children {
				indexes[index] = index
			}
		}
		for _, index := range indexes {
			if index < 0 || index >= len(arguments.Children) {
				continue
			}
			target := aliasExpressionSymbol(arguments.Children[index], semantics, eligible)
			if target != nil {
				function.aliasEvents[block] = append(function.aliasEvents[block], aliasEvent{offset: node.End, target: target})
			}
		}
	}
	for block := range function.aliasEvents {
		sort.SliceStable(function.aliasEvents[block], func(i, j int) bool {
			return function.aliasEvents[block][i].offset < function.aliasEvents[block][j].offset
		})
	}
	propagateAliases(function)
}

func aliasSymbol(tree *walk.Model, function *Function, symbol *semantic.Symbol) bool {
	if symbol == nil || symbol.Ambiguous || symbol.Function != function.Node || symbol.Decl == nil || symbol.Kind != semantic.SymbolLocal && symbol.Kind != semantic.SymbolParameter {
		return false
	}
	for _, child := range symbol.Decl.Children {
		if child.Kind == parser.KindDimension {
			return false
		}
	}
	if symbol.Kind == semantic.SymbolLocal && walk.HasChildToken(tree.Parent(symbol.Decl), token.KwStatic) {
		return false
	}
	if symbol.Kind == semantic.SymbolParameter {
		return true
	}
	return function.Block(symbol.Decl) != nil
}

func aliasExpressionSymbol(node *parser.Node, semantics *semantic.Model, eligible map[*semantic.Symbol]bool) *semantic.Symbol {
	for node != nil {
		switch node.Kind {
		case parser.KindIdentifier:
			symbol := semantics.Resolve(node)
			if eligible[symbol] {
				return symbol
			}
			return nil
		case parser.KindParenthesizedExpression, parser.KindTaggedExpression:
			node = node.Field("expression")
		default:
			return nil
		}
	}
	return nil
}

func propagateAliases(function *Function) {
	available := map[*Block]bool{function.Entry: true}
	aliasOut := make(map[*Block]aliasState)
	function.aliasIn[function.Entry] = initialAliasState(function.aliasIndexes)
	queue := []*Block{function.Entry}
	queued := map[*Block]bool{function.Entry: true}
	for len(queue) != 0 {
		block := queue[0]
		queue = queue[1:]
		queued[block] = false
		out := copyAliasState(function.aliasIn[block])
		applyAliasEvents(out, function.aliasEvents[block], -1, function.aliasIndexes)
		if sameAliasState(aliasOut[block], out) && aliasOut[block] != nil {
			continue
		}
		aliasOut[block] = out
		for _, edge := range block.Successors {
			incoming, ok := joinedPredecessorAliases(edge.To, available, aliasOut, function.aliasIndexes)
			if !ok {
				continue
			}
			changed := !available[edge.To] || !sameAliasState(function.aliasIn[edge.To], incoming)
			available[edge.To] = true
			function.aliasIn[edge.To] = incoming
			if changed && !queued[edge.To] {
				queue = append(queue, edge.To)
				queued[edge.To] = true
			}
		}
	}
}

func applyAliasEvents(state aliasState, events []aliasEvent, until int, indexes map[*semantic.Symbol]int) {
	for _, event := range events {
		if until >= 0 && event.offset > until {
			break
		}
		if event.target == event.source {
			continue
		}
		detachAlias(state, event.target, indexes)
		if event.source == nil {
			continue
		}
		class := state[event.source]
		if indexes[event.target] < class {
			for symbol, current := range state {
				if current == class {
					state[symbol] = indexes[event.target]
				}
			}
			class = indexes[event.target]
		}
		state[event.target] = class
	}
}

func detachAlias(state aliasState, symbol *semantic.Symbol, indexes map[*semantic.Symbol]int) {
	class := state[symbol]
	state[symbol] = indexes[symbol]
	if class != indexes[symbol] {
		return
	}
	replacement := 0
	for candidate, current := range state {
		if candidate != symbol && current == class && (replacement == 0 || indexes[candidate] < replacement) {
			replacement = indexes[candidate]
		}
	}
	if replacement == 0 {
		return
	}
	for candidate, current := range state {
		if candidate != symbol && current == class {
			state[candidate] = replacement
		}
	}
}

func joinedPredecessorAliases(block *Block, available map[*Block]bool, aliasOut map[*Block]aliasState, indexes map[*semantic.Symbol]int) (aliasState, bool) {
	var result aliasState
	found := false
	for _, predecessor := range block.Predecessors {
		if !available[predecessor] || aliasOut[predecessor] == nil {
			continue
		}
		if !found {
			result = copyAliasState(aliasOut[predecessor])
			found = true
			continue
		}
		result = intersectAliasStates(result, aliasOut[predecessor], indexes)
	}
	return result, found
}

func intersectAliasStates(left, right aliasState, indexes map[*semantic.Symbol]int) aliasState {
	type pair struct {
		left  int
		right int
	}
	classes := make(map[pair]int, len(indexes))
	for symbol, index := range indexes {
		key := pair{left: left[symbol], right: right[symbol]}
		if current := classes[key]; current == 0 || index < current {
			classes[key] = index
		}
	}
	result := make(aliasState, len(indexes))
	for symbol := range indexes {
		result[symbol] = classes[pair{left: left[symbol], right: right[symbol]}]
	}
	return result
}

func initialAliasState(indexes map[*semantic.Symbol]int) aliasState {
	result := make(aliasState, len(indexes))
	for symbol, index := range indexes {
		result[symbol] = index
	}
	return result
}

func copyAliasState(state aliasState) aliasState {
	result := make(aliasState, len(state))
	for symbol, class := range state {
		result[symbol] = class
	}
	return result
}

func sameAliasState(left, right aliasState) bool {
	if len(left) != len(right) {
		return false
	}
	for symbol, class := range left {
		if right[symbol] != class {
			return false
		}
	}
	return true
}
