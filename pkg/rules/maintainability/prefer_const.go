package maintainability

import (
	"fmt"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type PreferConst struct{}

func (PreferConst) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "prefer-const",
		Name:            "Prefer const",
		Summary:         "Reports initialized local variables that are never modified",
		Explanation:     "Initialized local scalar variables should be const when every use is read-only. Unused variables, static declarations, arrays, unresolved call arguments, and uncertain syntax are ignored.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"const", "variables", "semantic"},
	}
}

func (PreferConst) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, symbol := range ctx.Semantic.Symbols {
		if !preferConstCandidate(ctx, symbol) {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("local variable %q is never modified and can be declared const", symbol.Name),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(symbol.NameNode),
		})
	}
}

func preferConstCandidate(ctx *lint.Context, symbol *semantic.Symbol) bool {
	if symbol.Kind != semantic.SymbolLocal || symbol.Ambiguous || symbol.Constant || strings.HasPrefix(symbol.Name, "_") {
		return false
	}
	declaration := ctx.Walk.Parent(symbol.Decl)
	if symbol.Decl.HasError || declaration == nil || declaration.HasError || ctx.Walk.Inactive(symbol.Decl) || ctx.Walk.Uncertain(symbol.Decl) {
		return false
	}
	if symbol.Decl.Field("initializer") == nil || walk.HasChildToken(declaration, token.KwStatic) || preferConstHasDimension(symbol.Decl) {
		return false
	}
	read := false
	for _, reference := range ctx.Semantic.References(symbol) {
		if reference.Node.HasError || ctx.Walk.Inactive(reference.Node) || ctx.Walk.Uncertain(reference.Node) || preferConstInMacro(ctx, reference.Node) {
			return false
		}
		switch reference.Kind {
		case semantic.ReferenceWrite, semantic.ReferenceReadWrite:
			return false
		case semantic.ReferenceRead:
			if !preferConstCallArgumentReadOnly(ctx, reference.Node) {
				return false
			}
			read = true
		}
	}
	return read
}

func preferConstHasDimension(declarator *parser.Node) bool {
	for _, child := range declarator.Children {
		if child.Kind == parser.KindDimension {
			return true
		}
	}
	return false
}

func preferConstInMacro(ctx *lint.Context, node *parser.Node) bool {
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindMacroInvocation, parser.KindMacroInvocationBlock, parser.KindMacroBody:
			return true
		case parser.KindFunctionDefinition:
			return false
		}
	}
	return false
}

func preferConstCallArgumentReadOnly(ctx *lint.Context, node *parser.Node) bool {
	current := node
	for parent := ctx.Walk.Parent(current); parent != nil; parent = ctx.Walk.Parent(current) {
		if parent.Kind == parser.KindArgumentList {
			return preferConstParameterReadOnly(ctx, parent, current)
		}
		if parent.Kind == parser.KindFunctionDefinition || parent.Kind == parser.KindBlock {
			return true
		}
		current = parent
	}
	return true
}

func preferConstParameterReadOnly(ctx *lint.Context, arguments, argument *parser.Node) bool {
	index := -1
	for current, child := range arguments.Children {
		if child == argument {
			index = current
			break
		}
	}
	if index < 0 || preferConstContainsKind(argument, parser.KindArgumentName) {
		return false
	}
	call := ctx.Walk.Parent(arguments)
	if call == nil || call.Kind != parser.KindCallExpression {
		return false
	}
	callee := call.Field("function")
	if callee == nil {
		callee = call.Field("callee")
	}
	if callee == nil || callee.Kind != parser.KindIdentifier {
		return false
	}
	name := ctx.Walk.Text(callee)
	if native, ok := ctx.Natives()[name]; ok {
		parameter, found := preferConstAPIParameter(native.Parameters, index)
		return found && preferConstAPIParameterReadOnly(parameter)
	}
	if function, ok := ctx.Functions()[name]; ok {
		parameter, found := preferConstAPIParameter(function.Parameters, index)
		return found && preferConstAPIParameterReadOnly(parameter)
	}
	symbol := ctx.Semantic.Resolve(callee)
	if symbol == nil || symbol.Ambiguous {
		return false
	}
	parameter, found := preferConstFunctionParameter(symbol.Decl, index)
	if !found || parameter.HasError || ctx.Walk.Uncertain(parameter) {
		return false
	}
	return !walk.ReferencesByAmpersand(ctx.File.Parsed.Tokens, parameter) || walk.HasChildToken(parameter, token.KwConst)
}

func preferConstAPIParameter(parameters []api.Parameter, index int) (api.Parameter, bool) {
	if index < len(parameters) {
		return parameters[index], true
	}
	if len(parameters) != 0 && parameters[len(parameters)-1].Variadic {
		return parameters[len(parameters)-1], true
	}
	return api.Parameter{}, false
}

func preferConstAPIParameterReadOnly(parameter api.Parameter) bool {
	return !parameter.Output && (!parameter.Reference || parameter.Const)
}

func preferConstFunctionParameter(function *parser.Node, index int) (*parser.Node, bool) {
	if function == nil {
		return nil, false
	}
	parameters := function.Field("parameters")
	if parameters == nil || len(parameters.Children) == 0 {
		return nil, false
	}
	if index < len(parameters.Children) {
		return parameters.Children[index], true
	}
	last := parameters.Children[len(parameters.Children)-1]
	if last.Tok.Kind == token.Ellipsis || walk.HasChildToken(last, token.Ellipsis) {
		return last, true
	}
	return nil, false
}

func preferConstContainsKind(root *parser.Node, kind parser.Kind) bool {
	if root == nil {
		return false
	}
	if root.Kind == kind {
		return true
	}
	for _, child := range root.Children {
		if preferConstContainsKind(child, kind) {
			return true
		}
	}
	return false
}
