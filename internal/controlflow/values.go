package controlflow

import (
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

type valueEventKind uint8

const (
	valueInvalidate valueEventKind = iota
	valueAssign
)

type valueEvent struct {
	kind   valueEventKind
	offset int
	symbol *semantic.Symbol
	expr   *parser.Node
	zero   bool
}

func buildValueFlow(function *Function, tree *walk.Model, semantics *semantic.Model, options Options, nodes *functionNodes, symbols []*semantic.Symbol) {
	function.valueIn = make(map[*Block]map[*semantic.Symbol]int64)
	function.valueEvents = make(map[*Block][]valueEvent)
	if function.Uncertain || semantics == nil {
		return
	}
	tracked := make(map[*semantic.Symbol]struct{})
	for _, symbol := range symbols {
		if valueSymbol(tree, function, symbol) {
			tracked[symbol] = struct{}{}
			addDeclarationValueEvent(function, symbol)
		}
	}
	for _, node := range nodes.assignments {
		left := node.Field("left")
		if left == nil || left.Kind != parser.KindIdentifier {
			continue
		}
		symbol := semantics.Resolve(left)
		if _, ok := tracked[symbol]; !ok {
			continue
		}
		block := function.Block(node)
		if block == nil {
			continue
		}
		event := valueEvent{kind: valueInvalidate, offset: node.End, symbol: symbol}
		if node.Tok.Kind == token.Assign && (block.Node == nil || block.Node.Kind != parser.KindForStatement) && !conditionallyExecuted(tree, node) {
			event.kind = valueAssign
			event.expr = node.Field("right")
		}
		function.valueEvents[block] = append(function.valueEvents[block], event)
	}
	for _, node := range nodes.updates {
		expression := node.Field("expression")
		if expression == nil || expression.Kind != parser.KindIdentifier {
			continue
		}
		symbol := semantics.Resolve(expression)
		if _, ok := tracked[symbol]; !ok {
			continue
		}
		block := function.Block(node)
		if block != nil {
			function.valueEvents[block] = append(function.valueEvents[block], valueEvent{kind: valueInvalidate, offset: node.End, symbol: symbol})
		}
	}
	for _, node := range nodes.calls {
		block := function.Block(node)
		if block == nil {
			continue
		}
		arguments := node.Field("arguments")
		if arguments == nil {
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
		seen := make(map[*semantic.Symbol]struct{})
		for _, index := range indexes {
			if index < 0 || index >= len(arguments.Children) {
				continue
			}
			symbol := callArgumentSymbol(arguments.Children[index], semantics)
			if _, tracked := tracked[symbol]; tracked {
				seen[symbol] = struct{}{}
			}
		}
		for symbol := range seen {
			function.valueEvents[block] = append(function.valueEvents[block], valueEvent{kind: valueInvalidate, offset: node.End, symbol: symbol})
		}
	}
	for block := range function.valueEvents {
		sort.SliceStable(function.valueEvents[block], func(i, j int) bool {
			if function.valueEvents[block][i].offset != function.valueEvents[block][j].offset {
				return function.valueEvents[block][i].offset < function.valueEvents[block][j].offset
			}
			return function.valueEvents[block][i].kind < function.valueEvents[block][j].kind
		})
	}
	propagateValues(function, semantics)
}

func callEffects(call *parser.Node, semantics *semantic.Model, options Options) (CallEffects, bool) {
	if options.ResolveCallEffects != nil {
		if effects, known := options.ResolveCallEffects(call); known {
			return effects, true
		}
	}
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier {
		return CallEffects{}, false
	}
	symbol := semantics.Resolve(callee)
	if symbol == nil || symbol.Kind != semantic.SymbolFunction || symbol.Ambiguous || symbol.Decl == nil {
		return CallEffects{}, false
	}
	parameters := symbol.Decl.Field("parameters")
	if parameters == nil {
		return CallEffects{Complete: true}, true
	}
	effects := CallEffects{Complete: true}
	index := 0
	for _, parameter := range parameters.Children {
		if parameter.Kind != parser.KindParameter {
			continue
		}
		if walk.ReferencesByAmpersand(semantics.File.Tokens, parameter) || hasCallEffectDimension(parameter) {
			effects.MutatedArguments = append(effects.MutatedArguments, index)
		}
		index++
	}
	return effects, true
}

func callArgumentSymbol(node *parser.Node, semantics *semantic.Model) *semantic.Symbol {
	for node != nil {
		switch node.Kind {
		case parser.KindIdentifier:
			return semantics.Resolve(node)
		case parser.KindParenthesizedExpression, parser.KindTaggedExpression:
			node = node.Field("expression")
		default:
			return nil
		}
	}
	return nil
}

func hasCallEffectDimension(node *parser.Node) bool {
	for _, child := range node.Children {
		if child.Kind == parser.KindDimension {
			return true
		}
	}
	return false
}

func valueSymbol(tree *walk.Model, function *Function, symbol *semantic.Symbol) bool {
	if symbol == nil || symbol.Ambiguous || symbol.Kind != semantic.SymbolLocal || symbol.Function != function.Node || symbol.Decl == nil {
		return false
	}
	if len(symbol.Tags) != 0 && (len(symbol.Tags) != 1 || symbol.Tags[0] != "_") {
		return false
	}
	for _, child := range symbol.Decl.Children {
		if child.Kind == parser.KindDimension {
			return false
		}
	}
	declaration := tree.Parent(symbol.Decl)
	if declaration == nil {
		return false
	}
	for _, child := range declaration.Children {
		if child.Tok.Kind == token.KwStatic {
			return false
		}
	}
	block := function.Block(symbol.Decl)
	return block != nil && (block.Node == nil || block.Node.Kind != parser.KindForStatement)
}

func conditionallyExecuted(tree *walk.Model, node *parser.Node) bool {
	for parent := tree.Parent(node); parent != nil; parent = tree.Parent(parent) {
		switch parent.Kind {
		case parser.KindTernaryExpression:
			condition := parent.Field("condition")
			if condition == nil || node.Start < condition.Start || node.End > condition.End {
				return true
			}
		case parser.KindBinaryExpression:
			if parent.Tok.Kind == token.AndAnd || parent.Tok.Kind == token.OrOr {
				right := parent.Field("right")
				if right != nil && node.Start >= right.Start && node.End <= right.End {
					return true
				}
			}
		}
		if walk.IsStatement(parent) {
			return false
		}
	}
	return false
}

func addDeclarationValueEvent(function *Function, symbol *semantic.Symbol) {
	block := function.Block(symbol.Decl)
	event := valueEvent{kind: valueAssign, offset: symbol.Decl.End, symbol: symbol, expr: symbol.Decl.Field("initializer")}
	if event.expr == nil {
		event.zero = true
	}
	function.valueEvents[block] = append(function.valueEvents[block], event)
}

func propagateValues(function *Function, semantics *semantic.Model) {
	available := make(map[*Block]bool)
	valueOut := make(map[*Block]map[*semantic.Symbol]int64)
	available[function.Entry] = true
	function.valueIn[function.Entry] = make(map[*semantic.Symbol]int64)
	queue := []*Block{function.Entry}
	queued := map[*Block]bool{function.Entry: true}
	for len(queue) != 0 {
		block := queue[0]
		queue = queue[1:]
		queued[block] = false
		out := copyValueState(function.valueIn[block])
		applyValueEvents(out, function.valueEvents[block], -1, semantics)
		if valueStatesEqual(valueOut[block], out) && valueOut[block] != nil {
			continue
		}
		valueOut[block] = out
		for _, edge := range block.Successors {
			successor := edge.To
			incoming, ok := joinedPredecessorValues(successor, available, valueOut)
			if !ok {
				continue
			}
			changed := !available[successor] || !valueStatesEqual(function.valueIn[successor], incoming)
			available[successor] = true
			function.valueIn[successor] = incoming
			if changed && !queued[successor] {
				queue = append(queue, successor)
				queued[successor] = true
			}
		}
	}
}

func joinedPredecessorValues(block *Block, available map[*Block]bool, valueOut map[*Block]map[*semantic.Symbol]int64) (map[*semantic.Symbol]int64, bool) {
	var result map[*semantic.Symbol]int64
	found := false
	for _, predecessor := range block.Predecessors {
		if !available[predecessor] || valueOut[predecessor] == nil {
			continue
		}
		if !found {
			result = copyValueState(valueOut[predecessor])
			found = true
			continue
		}
		for symbol, value := range result {
			other, ok := valueOut[predecessor][symbol]
			if !ok || other != value {
				delete(result, symbol)
			}
		}
	}
	return result, found
}

func applyValueEvents(state map[*semantic.Symbol]int64, events []valueEvent, until int, semantics *semantic.Model) {
	for _, event := range events {
		if until >= 0 && event.offset > until {
			break
		}
		if event.kind == valueInvalidate {
			delete(state, event.symbol)
			continue
		}
		if event.zero {
			state[event.symbol] = 0
			continue
		}
		value, ok := semantics.EvalWithValues(event.expr, state)
		if !ok {
			delete(state, event.symbol)
			continue
		}
		state[event.symbol] = value
	}
}

func copyValueState(state map[*semantic.Symbol]int64) map[*semantic.Symbol]int64 {
	result := make(map[*semantic.Symbol]int64, len(state))
	for symbol, value := range state {
		result[symbol] = value
	}
	return result
}

func valueStatesEqual(left, right map[*semantic.Symbol]int64) bool {
	if len(left) != len(right) {
		return false
	}
	for symbol, value := range left {
		other, ok := right[symbol]
		if !ok || other != value {
			return false
		}
	}
	return true
}
