package openmp

import (
	"sort"
	"strconv"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type resourceAcquisition struct {
	call    *parser.Node
	write   *parser.Node
	symbol  *semantic.Symbol
	block   *controlflow.Block
	name    string
	release string
}

type resourceCallable struct {
	name       string
	returnTag  string
	parameters []api.Parameter
	release    string
	mustUse    bool
	project    bool
}

func calledResourceFunction(ctx *lint.Context, call *parser.Node) (resourceCallable, bool) {
	return calledResourceFunctionIn(ctx, ctx.ProjectFile, ctx.Walk, ctx.Semantic, call, make(map[string]bool))
}

func calledResourceFunctionIn(ctx *lint.Context, file *project.File, tree *walk.Model, semantics *semantic.Model, call *parser.Node, visiting map[string]bool) (resourceCallable, bool) {
	if call == nil || call.HasError || tree == nil || semantics == nil || tree.Uncertain(call) {
		return resourceCallable{}, false
	}
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier {
		return resourceCallable{}, false
	}
	name := tree.Text(callee)
	if projectDefinesName(ctx, name) {
		return resourceCallable{}, false
	}
	if native, known := ctx.Natives()[name]; known {
		if symbol := semantics.Resolve(callee); symbol != nil && !walk.HasChildToken(symbol.Decl, token.KwNative) {
			return resourceCallable{}, false
		}
		return resourceCallable{name: name, returnTag: native.ReturnTag, parameters: native.Parameters, release: native.Release, mustUse: native.MustUse}, true
	}
	if contract, known := ctx.Functions()[name]; known {
		projectCallable := false
		if symbol := semantics.Resolve(callee); symbol != nil {
			if symbol.Ambiguous || symbol.Kind != semantic.SymbolFunction || symbol.Decl == nil || walk.HasChildToken(symbol.Decl, token.KwNative) {
				return resourceCallable{}, false
			}
			projectCallable = true
		} else if ctx.Project != nil && file != nil {
			variants := ctx.Project.FunctionVariants(file, callee)
			projectCallable = len(variants) != 0
			if !projectCallable {
				for _, declaration := range ctx.Project.Declarations[name] {
					if declaration.Kind == semantic.SymbolFunction {
						return resourceCallable{}, false
					}
				}
			}
		}
		return resourceCallable{name: name, returnTag: contract.ReturnTag, parameters: contract.Parameters, release: contract.Release, project: projectCallable}, true
	}
	if ctx.Project == nil || file == nil {
		return resourceCallable{}, false
	}
	variants := ctx.Project.FunctionVariants(file, callee)
	if len(variants) == 0 {
		return resourceCallable{}, false
	}
	var inferred resourceCallable
	for index, declaration := range variants {
		current, ok := inferredResourceReturn(ctx, declaration, visiting)
		if !ok || index != 0 && (current.release != inferred.release || current.returnTag != inferred.returnTag) {
			return resourceCallable{}, false
		}
		inferred = current
	}
	inferred.name = name
	inferred.project = true
	return inferred, true
}

func inferredResourceReturn(ctx *lint.Context, declaration project.Declaration, visiting map[string]bool) (resourceCallable, bool) {
	if declaration.File == nil || declaration.Node == nil || declaration.Node.Kind != parser.KindFunctionDefinition || declaration.Symbol == nil || declaration.Symbol.Ambiguous {
		return resourceCallable{}, false
	}
	key := declaration.File.Path + ":" + strconv.Itoa(declaration.Node.Start)
	if visiting[key] {
		return resourceCallable{}, false
	}
	visiting[key] = true
	defer delete(visiting, key)
	body := declaration.Node.Field("body")
	if body == nil || len(body.Children) != 1 {
		return resourceCallable{}, false
	}
	statement := body.Children[0]
	if statement.Kind != parser.KindReturnStatement {
		return resourceCallable{}, false
	}
	value := unwrapParentheses(statement.Field("value"))
	if value == nil || value.Kind != parser.KindCallExpression {
		return resourceCallable{}, false
	}
	callable, ok := calledResourceFunctionIn(ctx, declaration.File, declaration.File.Walk, declaration.File.Semantic, value, visiting)
	if !ok || callable.release == "" {
		return resourceCallable{}, false
	}
	return resourceCallable{returnTag: callable.returnTag, release: callable.release}, true
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
	callable, ok := calledResourceFunction(ctx, call)
	function := ctx.Flow.Function(symbol.Function)
	if !ok || callable.release == "" || function == nil || function.Uncertain {
		return resourceAcquisition{}, false
	}
	block := function.Block(call)
	if !function.ReachableBlock(block) {
		return resourceAcquisition{}, false
	}
	return resourceAcquisition{call: call, write: symbol.NameNode, symbol: symbol, block: block, name: callable.name, release: callable.release}, true
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
	callable, ok := calledResourceFunction(ctx, call)
	function := ctx.Flow.Function(symbol.Function)
	if !ok || callable.release == "" || function == nil || function.Uncertain {
		return resourceAcquisition{}, false
	}
	block := function.Block(assignment)
	if !function.ReachableBlock(block) {
		return resourceAcquisition{}, false
	}
	return resourceAcquisition{call: call, write: left, symbol: symbol, block: block, name: callable.name, release: callable.release}, true
}

func resourceReferencesByBlock(ctx *lint.Context, function *controlflow.Function, symbol *semantic.Symbol) map[*controlflow.Block][]semantic.Reference {
	references := make(map[*controlflow.Block][]semantic.Reference)
	seen := make(map[*parser.Node]bool)
	for _, candidate := range ctx.Semantic.Symbols {
		if candidate == nil || candidate.Function != symbol.Function {
			continue
		}
		for _, reference := range ctx.Semantic.References(candidate) {
			if seen[reference.Node] || !resourceAliasesAt(ctx, reference.Node, symbol, candidate) {
				continue
			}
			block := function.Block(reference.Node)
			if block != nil {
				seen[reference.Node] = true
				references[block] = append(references[block], reference)
			}
		}
	}
	for block := range references {
		sort.Slice(references[block], func(i, j int) bool {
			return references[block][i].Node.Start < references[block][j].Node.Start
		})
	}
	return references
}

func resourceAliasesAt(ctx *lint.Context, node *parser.Node, left, right *semantic.Symbol) bool {
	if left == right {
		return true
	}
	for _, alias := range ctx.Flow.Aliases(node, left) {
		if alias == right {
			return true
		}
	}
	return false
}

func resourceCallTransfers(ctx *lint.Context, call, reference *parser.Node, releaser string, visiting map[string]bool) bool {
	arguments := call.Field("arguments")
	index := resourceArgumentIndex(arguments, reference)
	if index < 0 || hasNamedArgument(arguments) {
		return false
	}
	if callable, known := calledResourceFunction(ctx, call); known {
		if callable.name == releaser {
			return true
		}
		if index < len(callable.parameters) && callable.parameters[index].Ownership == "transferred" {
			return true
		}
	}
	if ctx.Project == nil || ctx.ProjectFile == nil {
		return false
	}
	callee := call.Field("function")
	variants := ctx.Project.FunctionVariants(ctx.ProjectFile, callee)
	if len(variants) == 0 {
		return false
	}
	for _, declaration := range variants {
		if !inferredResourceTransfer(ctx, declaration, index, releaser, visiting) {
			return false
		}
	}
	return true
}

func resourceArgumentIndex(arguments, reference *parser.Node) int {
	if arguments == nil {
		return -1
	}
	for index, argument := range arguments.Children {
		if nodeInField(reference, argument) {
			return index
		}
	}
	return -1
}

func inferredResourceTransfer(ctx *lint.Context, declaration project.Declaration, parameterIndex int, releaser string, visiting map[string]bool) bool {
	if declaration.File == nil || declaration.Node == nil || declaration.Node.Kind != parser.KindFunctionDefinition || declaration.Symbol == nil || declaration.Symbol.Ambiguous {
		return false
	}
	parameters := resourceParameters(declaration)
	if parameterIndex < 0 || parameterIndex >= len(parameters) {
		return false
	}
	key := declaration.File.Path + ":" + strconv.Itoa(declaration.Node.Start) + ":" + strconv.Itoa(parameterIndex) + ":" + releaser
	if visiting[key] {
		return false
	}
	visiting[key] = true
	defer delete(visiting, key)
	inner := topLevelResourceCall(declaration)
	if inner == nil {
		return false
	}
	arguments := inner.Field("arguments")
	innerIndex := -1
	for index, argument := range arguments.Children {
		expression := unwrapParentheses(argument)
		if expression != nil && expression.Kind == parser.KindIdentifier && declaration.File.Semantic.Resolve(expression) == parameters[parameterIndex] {
			innerIndex = index
			break
		}
	}
	if innerIndex < 0 || hasNamedArgument(arguments) {
		return false
	}
	callee := inner.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier {
		return false
	}
	name := declaration.File.Walk.Text(callee)
	if name == releaser {
		return true
	}
	if callable, known := calledResourceFunctionIn(ctx, declaration.File, declaration.File.Walk, declaration.File.Semantic, inner, visiting); known && innerIndex < len(callable.parameters) && callable.parameters[innerIndex].Ownership == "transferred" {
		return true
	}
	variants := ctx.Project.FunctionVariants(declaration.File, callee)
	if len(variants) == 0 {
		return false
	}
	for _, variant := range variants {
		if !inferredResourceTransfer(ctx, variant, innerIndex, releaser, visiting) {
			return false
		}
	}
	return true
}

func resourceParameters(declaration project.Declaration) []*semantic.Symbol {
	var parameters []*semantic.Symbol
	for _, symbol := range declaration.File.Semantic.Symbols {
		if symbol.Kind == semantic.SymbolParameter && symbol.Function == declaration.Node && !symbol.Ambiguous {
			parameters = append(parameters, symbol)
		}
	}
	sort.Slice(parameters, func(i, j int) bool {
		return parameters[i].NameNode.Start < parameters[j].NameNode.Start
	})
	return parameters
}

func topLevelResourceCall(declaration project.Declaration) *parser.Node {
	body := declaration.Node.Field("body")
	if body == nil || len(body.Children) != 1 {
		return nil
	}
	statement := body.Children[0]
	var expression *parser.Node
	switch statement.Kind {
	case parser.KindExpressionStatement:
		expression = statement.Field("expression")
	case parser.KindReturnStatement:
		expression = statement.Field("value")
	}
	expression = unwrapParentheses(expression)
	if expression == nil || expression.Kind != parser.KindCallExpression {
		return nil
	}
	return expression
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
			if parent.Tok.Kind == token.Assign && left != nil && right != nil && left.Kind == parser.KindIdentifier && right.Kind == parser.KindIdentifier {
				leftSymbol := ctx.Semantic.Resolve(left)
				rightSymbol := ctx.Semantic.Resolve(right)
				if leftSymbol != nil && rightSymbol != nil && resourceAliasesAt(ctx, reference.Node, symbol, rightSymbol) && (leftSymbol.Kind == semantic.SymbolLocal || leftSymbol.Kind == semantic.SymbolParameter) {
					return false
				}
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
		callable, known := calledResourceFunction(ctx, parent)
		if !known {
			return true
		}
		if callable.name == releaser {
			return true
		}
		if hasNamedArgument(arguments) {
			return true
		}
		for index, argument := range arguments.Children {
			if !nodeInField(reference.Node, argument) || index >= len(callable.parameters) {
				continue
			}
			parameter := callable.parameters[index]
			if parameter.Reference || parameter.Ownership == "transferred" || callable.project && parameter.Ownership != "borrowed" {
				return true
			}
			return false
		}
		return true
	}
	return false
}

func nodeInField(node, field *parser.Node) bool {
	return node != nil && field != nil && node.Start >= field.Start && node.End <= field.End
}
