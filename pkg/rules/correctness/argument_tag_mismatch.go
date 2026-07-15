package correctness

import (
	"fmt"
	"strconv"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type ArgumentTagMismatch struct{}

type argumentTagParameter struct {
	tags     []string
	variadic bool
	known    bool
}

func (ArgumentTagMismatch) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "argument-tag-mismatch",
		Name:            "Argument tag mismatch",
		Summary:         "Reports arguments incompatible with definite parameter tags",
		Explanation:     "Calls to resolved project functions and known APIs are checked for representation-changing Float mismatches and conflicts between distinct tags. Untagged non-Float values and zero are representation-compatible. Union, variadic, named, unresolved, macro-derived, uncertain, malformed, and structured-field arguments are ignored.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"arguments", "tags", "calls", "project", "api"},
	}
}

func (ArgumentTagMismatch) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, call := range ctx.Walk.OfKind(parser.KindCallExpression) {
		if argumentTagSkipped(ctx, call) {
			continue
		}
		name, parameters, ok := argumentTagSignature(ctx, call)
		if !ok {
			continue
		}
		arguments := call.Field("arguments")
		if arguments == nil || unitNodeHasError(arguments) {
			continue
		}
		for index, argument := range arguments.Children {
			if index >= len(parameters) || parameters[index].variadic {
				break
			}
			if argumentTagNamed(argument) || !parameters[index].known || argumentTagSkipped(ctx, argument) {
				continue
			}
			actual, known := argumentExpressionTags(ctx, argument)
			if !known || argumentTagsCompatible(ctx, argument, parameters[index].tags, actual) {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  argumentTagMessage(name, index+1, parameters[index].tags, actual),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(argument),
			})
		}
	}
}

func argumentTagSignature(ctx *lint.Context, call *parser.Node) (string, []argumentTagParameter, bool) {
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier {
		return "", nil, false
	}
	name := ctx.Walk.Text(callee)
	if ctx.Project != nil && ctx.ProjectFile != nil {
		if declaration, ok := ctx.Project.Resolve(ctx.ProjectFile, callee); ok {
			if declaration.Kind != semantic.SymbolFunction || declaration.Node == nil || declaration.Symbol == nil || declaration.Symbol.Ambiguous {
				return "", nil, false
			}
			return name, argumentSourceParameters(declaration.File.Semantic, declaration.Node), true
		}
		for _, declaration := range ctx.Project.Declarations[name] {
			if declaration.Kind == semantic.SymbolFunction {
				return "", nil, false
			}
		}
	}
	if symbol := ctx.Semantic.ResolveAsCallTarget(callee); symbol != nil {
		if symbol.Kind != semantic.SymbolFunction || symbol.Decl == nil || symbol.Ambiguous {
			return "", nil, false
		}
		return name, argumentSourceParameters(ctx.Semantic, symbol.Decl), true
	}
	if native, ok := ctx.Natives()[name]; ok {
		return name, argumentAPIParameters(native.Parameters), true
	}
	if function, ok := ctx.Functions()[name]; ok {
		return name, argumentAPIParameters(function.Parameters), true
	}
	return "", nil, false
}

func argumentSourceParameters(model *semantic.Model, function *parser.Node) []argumentTagParameter {
	list := function.Field("parameters")
	if list == nil {
		return nil
	}
	var result []argumentTagParameter
	for _, parameter := range list.Children {
		if parameter.Kind != parser.KindParameter {
			continue
		}
		if parameter.Field("name") == nil && parameter.Tok.Kind == token.Ellipsis {
			result = append(result, argumentTagParameter{variadic: true, known: true})
			continue
		}
		item := argumentTagParameter{tags: []string{""}}
		for _, symbol := range model.Symbols {
			if symbol.Kind != semantic.SymbolParameter || symbol.Decl != parameter {
				continue
			}
			if !symbol.Ambiguous {
				item.known = true
				if len(symbol.Tags) != 0 {
					item.tags = normalizeArgumentTags(symbol.Tags)
				}
			}
			break
		}
		result = append(result, item)
	}
	return result
}

func argumentAPIParameters(parameters []api.Parameter) []argumentTagParameter {
	result := make([]argumentTagParameter, len(parameters))
	for index, parameter := range parameters {
		tags := []string{""}
		if parameter.Tag != "" && parameter.Tag != "_" {
			tags = []string{parameter.Tag}
		}
		result[index] = argumentTagParameter{tags: tags, variadic: parameter.Variadic, known: true}
	}
	return result
}

func argumentExpressionTags(ctx *lint.Context, node *parser.Node) ([]string, bool) {
	if node == nil || unitNodeHasError(node) || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return nil, false
	}
	if node.Kind == parser.KindSubscriptExpression {
		array := node.Field("array")
		if array == nil || array.Kind == parser.KindSubscriptExpression {
			return nil, false
		}
	}
	if tags := ctx.Semantic.ExpressionTags(node); len(tags) != 0 {
		return normalizeArgumentTags(tags), true
	}
	switch node.Kind {
	case parser.KindParenthesizedExpression:
		return argumentExpressionTags(ctx, node.Field("expression"))
	case parser.KindIdentifier:
		if ctx.Project != nil && ctx.ProjectFile != nil {
			if declaration, ok := ctx.Project.Resolve(ctx.ProjectFile, node); ok && declaration.Symbol != nil && !declaration.Symbol.Ambiguous {
				if len(declaration.Symbol.Tags) == 0 {
					return []string{""}, true
				}
				return normalizeArgumentTags(declaration.Symbol.Tags), true
			}
		}
		if symbol := ctx.Semantic.Resolve(node); symbol != nil && !symbol.Ambiguous && len(symbol.Tags) == 0 {
			return []string{""}, true
		}
	case parser.KindLiteral:
		switch node.Tok.Kind {
		case token.FloatLiteral:
			return []string{"Float"}, true
		case token.IntLiteral, token.CharLiteral, token.StringLiteral, token.PackedString, token.KwNull:
			return []string{""}, true
		}
	case parser.KindUnaryExpression, parser.KindUpdateExpression:
		return argumentExpressionTags(ctx, node.Field("expression"))
	case parser.KindBinaryExpression:
		left, leftOK := argumentExpressionTags(ctx, node.Field("left"))
		right, rightOK := argumentExpressionTags(ctx, node.Field("right"))
		if leftOK && rightOK && sameArgumentTags(left, right) {
			return left, true
		}
	case parser.KindTernaryExpression:
		left, leftOK := argumentExpressionTags(ctx, node.Field("consequence"))
		right, rightOK := argumentExpressionTags(ctx, node.Field("alternative"))
		if leftOK && rightOK && sameArgumentTags(left, right) {
			return left, true
		}
	case parser.KindCallExpression:
		callee := node.Field("function")
		if callee != nil && callee.Kind == parser.KindIdentifier {
			name := ctx.Walk.Text(callee)
			if native, ok := ctx.Natives()[name]; ok {
				return normalizeArgumentTags([]string{native.ReturnTag}), true
			}
			if function, ok := ctx.Functions()[name]; ok {
				return normalizeArgumentTags([]string{function.ReturnTag}), true
			}
		}
	case parser.KindSizeofExpression, parser.KindTagofExpression, parser.KindArrayLiteral, parser.KindStringConcat:
		return []string{""}, true
	}
	if _, known := ctx.Eval(node); known {
		return []string{""}, true
	}
	return nil, false
}

func normalizeArgumentTags(tags []string) []string {
	result := make([]string, len(tags))
	for index, tag := range tags {
		if tag != "_" {
			result[index] = tag
		}
	}
	return result
}

func argumentTagsCompatible(ctx *lint.Context, node *parser.Node, expected, actual []string) bool {
	for _, expectedTag := range expected {
		for _, actualTag := range actual {
			if expectedTag == actualTag {
				return true
			}
			if expectedTag == "" || actualTag == "" {
				other := expectedTag
				if other == "" {
					other = actualTag
				}
				if other != "Float" || argumentZeroRepresentation(ctx, node, actualTag == "Float") {
					return true
				}
			}
		}
	}
	return false
}

func argumentZeroRepresentation(ctx *lint.Context, node *parser.Node, floating bool) bool {
	for node != nil && (node.Kind == parser.KindParenthesizedExpression || node.Kind == parser.KindTaggedExpression) {
		node = node.Field("expression")
	}
	if node == nil {
		return false
	}
	if floating && node.Kind == parser.KindLiteral && node.Tok.Kind == token.FloatLiteral {
		value, err := strconv.ParseFloat(strings.ReplaceAll(ctx.Walk.Text(node), "_", ""), 64)
		return err == nil && value == 0
	}
	value, known := ctx.Eval(node)
	return known && value == 0
}

func sameArgumentTags(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func argumentTagMessage(name string, position int, expected, actual []string) string {
	return fmt.Sprintf("argument %d to %q expects %s, but has %s", position, name, formatArgumentTags(expected), formatArgumentTags(actual))
}

func formatArgumentTags(tags []string) string {
	if len(tags) == 1 {
		if tags[0] == "" {
			return "no tag"
		}
		return "tag " + tags[0]
	}
	values := make([]string, len(tags))
	for index, tag := range tags {
		if tag == "" {
			values[index] = "_"
		} else {
			values[index] = tag
		}
	}
	return "one of tags {" + strings.Join(values, ", ") + "}"
}

func argumentTagNamed(node *parser.Node) bool {
	return node != nil && node.Kind == parser.KindAssignmentExpression && node.Field("left") != nil && node.Field("left").Kind == parser.KindArgumentName
}

func argumentTagSkipped(ctx *lint.Context, node *parser.Node) bool {
	if node == nil || unitNodeHasError(node) || node.Tok.Origin != nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return true
	}
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindDirectiveDefine, parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock,
			parser.KindSharedConditional, parser.KindConditionalFunction, parser.KindConditionalSplice:
			return true
		}
	}
	return false
}
