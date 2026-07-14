package openmp

import (
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type resourceAcquisition struct {
	call    *parser.Node
	write   *parser.Node
	symbol  *semantic.Symbol
	block   *controlflow.Block
	name    string
	release string
}

func resourceAcquisitions(ctx *lint.Context) map[*semantic.Symbol][]resourceAcquisition {
	bySymbol := make(map[*semantic.Symbol][]resourceAcquisition)
	for _, symbol := range ctx.Semantic.Symbols {
		declaration := ctx.Walk.Parent(symbol.Decl)
		if symbol.Kind != semantic.SymbolLocal || symbol.Ambiguous || walk.HasChildToken(declaration, token.KwStatic) {
			continue
		}
		if acquisition, ok := declaredResourceAcquisition(ctx, symbol); ok {
			bySymbol[symbol] = append(bySymbol[symbol], acquisition)
		}
	}
	for _, assignment := range ctx.Walk.OfKind(parser.KindAssignmentExpression) {
		if acquisition, ok := assignedResourceAcquisition(ctx, assignment); ok {
			bySymbol[acquisition.symbol] = append(bySymbol[acquisition.symbol], acquisition)
		}
	}
	return bySymbol
}

func declaredResourceAcquisition(ctx *lint.Context, symbol *semantic.Symbol) (resourceAcquisition, bool) {
	call := unwrapParentheses(symbol.Decl.Field("initializer"))
	if call == nil || call.Kind != parser.KindCallExpression {
		return resourceAcquisition{}, false
	}
	native, name, ok := calledNative(ctx, call)
	function := ctx.Flow.Function(symbol.Function)
	if !ok || native.Release == "" || function == nil || function.Uncertain {
		return resourceAcquisition{}, false
	}
	block := function.Block(call)
	if !function.ReachableBlock(block) {
		return resourceAcquisition{}, false
	}
	return resourceAcquisition{call: call, write: symbol.NameNode, symbol: symbol, block: block, name: name, release: native.Release}, true
}

func assignedResourceAcquisition(ctx *lint.Context, assignment *parser.Node) (resourceAcquisition, bool) {
	if assignment.Tok.Kind != token.Assign || assignment.HasError || ctx.Walk.Uncertain(assignment) {
		return resourceAcquisition{}, false
	}
	statement := ctx.Walk.Parent(assignment)
	left := assignment.Field("left")
	call := unwrapParentheses(assignment.Field("right"))
	if statement == nil || statement.Kind != parser.KindExpressionStatement || statement.Field("expression") != assignment || left == nil || left.Kind != parser.KindIdentifier || call == nil || call.Kind != parser.KindCallExpression {
		return resourceAcquisition{}, false
	}
	symbol := ctx.Semantic.Resolve(left)
	if symbol == nil || symbol.Kind != semantic.SymbolLocal || symbol.Ambiguous || walk.HasChildToken(ctx.Walk.Parent(symbol.Decl), token.KwStatic) {
		return resourceAcquisition{}, false
	}
	native, name, ok := calledNative(ctx, call)
	function := ctx.Flow.Function(symbol.Function)
	if !ok || native.Release == "" || function == nil || function.Uncertain {
		return resourceAcquisition{}, false
	}
	block := function.Block(assignment)
	if !function.ReachableBlock(block) {
		return resourceAcquisition{}, false
	}
	return resourceAcquisition{call: call, write: left, symbol: symbol, block: block, name: name, release: native.Release}, true
}

func resourceReferencesByBlock(ctx *lint.Context, function *controlflow.Function, symbol *semantic.Symbol) map[*controlflow.Block][]semantic.Reference {
	references := make(map[*controlflow.Block][]semantic.Reference)
	for _, reference := range ctx.Semantic.References(symbol) {
		block := function.Block(reference.Node)
		if block != nil {
			references[block] = append(references[block], reference)
		}
	}
	for block := range references {
		sort.Slice(references[block], func(i, j int) bool {
			return references[block][i].Node.Start < references[block][j].Node.Start
		})
	}
	return references
}

func resourceOwnershipEscapes(ctx *lint.Context, symbol *semantic.Symbol, reference semantic.Reference, releaser string) bool {
	if reference.Kind == semantic.ReferenceWrite || reference.Kind == semantic.ReferenceReadWrite {
		assignment := ctx.Walk.Parent(reference.Node)
		if assignment != nil && assignment.Kind == parser.KindAssignmentExpression && assignment.Tok.Kind == token.Assign {
			right := unwrapParentheses(assignment.Field("right"))
			if right != nil && right.Kind == parser.KindIdentifier && ctx.Semantic.Resolve(right) == symbol {
				return false
			}
		}
		return true
	}
	for node := reference.Node; node != nil; node = ctx.Walk.Parent(node) {
		parent := ctx.Walk.Parent(node)
		if parent == nil {
			break
		}
		if parent.Kind == parser.KindReturnStatement {
			return true
		}
		if parent.Kind == parser.KindAssignmentExpression && nodeInField(node, parent.Field("right")) {
			left := unwrapParentheses(parent.Field("left"))
			right := unwrapParentheses(parent.Field("right"))
			if parent.Tok.Kind == token.Assign && left != nil && right != nil && left.Kind == parser.KindIdentifier && right.Kind == parser.KindIdentifier && ctx.Semantic.Resolve(left) == symbol && ctx.Semantic.Resolve(right) == symbol {
				return false
			}
			return true
		}
		if parent.Kind != parser.KindCallExpression {
			continue
		}
		arguments := parent.Field("arguments")
		if !nodeInField(reference.Node, arguments) {
			continue
		}
		native, name, known := calledNative(ctx, parent)
		if !known {
			return true
		}
		if name == releaser {
			return true
		}
		for index, argument := range arguments.Children {
			if !nodeInField(reference.Node, argument) || index >= len(native.Parameters) {
				continue
			}
			if native.Parameters[index].Reference {
				return true
			}
		}
		return false
	}
	return false
}

func nodeInField(node, field *parser.Node) bool {
	return node != nil && field != nil && node.Start >= field.Start && node.End <= field.End
}
