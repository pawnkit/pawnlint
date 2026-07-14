package correctness

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type ForwardSignatureMismatch struct{}

func (ForwardSignatureMismatch) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "forward-signature-mismatch",
		Name:            "Forward signature mismatch",
		Summary:         "Reports definitions that do not match their forward declaration",
		Explanation:     "A function definition must match its forward declaration. The rule compares the signature parts exposed by the parser and reports only definite differences.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"functions", "forward", "signature", "semantic"},
	}
}

func (ForwardSignatureMismatch) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	forwards := make(map[string][]*parser.Node)
	definitions := make(map[string][]*parser.Node)
	ctx.Walk.Iter(func(node *parser.Node) {
		if ctx.Walk.Uncertain(node) || node.Field("state") != nil {
			return
		}
		nameNode := node.Field("name")
		if nameNode == nil {
			return
		}
		name := ctx.Walk.Text(nameNode)
		switch node.Kind {
		case parser.KindFunctionDeclaration:
			if walk.HasChildToken(node, token.KwForward) {
				forwards[name] = append(forwards[name], node)
			}
		case parser.KindFunctionDefinition:
			definitions[name] = append(definitions[name], node)
		}
	})
	for name, declarations := range forwards {
		defs := definitions[name]
		if len(declarations) != 1 || len(defs) != 1 {
			continue
		}
		reason := signatureDifference(ctx, declarations[0], defs[0])
		if reason == "" {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "definition of " + quoteFunction(name) + " does not match its forward declaration: " + reason,
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(defs[0].Field("name")),
			Notes: []diagnostic.RelatedLocation{{
				Range:   ctx.Walk.Range(declarations[0].Field("name")),
				Message: "forward declaration is here",
			}},
		})
	}
}

func signatureDifference(ctx *lint.Context, declaration, definition *parser.Node) string {
	if tagsKey(ctx, declaration) != tagsKey(ctx, definition) {
		return "return tag differs"
	}
	declParams := parameters(declaration)
	defParams := parameters(definition)
	if len(declParams) != len(defParams) {
		return fmt.Sprintf("parameter count differs (%d declared, %d defined)", len(declParams), len(defParams))
	}
	for i := range declParams {
		position := i + 1
		if variadic(declParams[i]) != variadic(defParams[i]) {
			return fmt.Sprintf("parameter %d variadic form differs", position)
		}
		if tagsKey(ctx, declParams[i]) != tagsKey(ctx, defParams[i]) {
			return fmt.Sprintf("parameter %d tag differs", position)
		}
		declName := declParams[i].Field("name")
		defName := defParams[i].Field("name")
		if declName != nil && defName != nil && ctx.Walk.Text(declName) != ctx.Walk.Text(defName) {
			return fmt.Sprintf("parameter %d name differs", position)
		}
		if walk.HasChildToken(declParams[i], token.KwConst) != walk.HasChildToken(defParams[i], token.KwConst) {
			return fmt.Sprintf("parameter %d const qualifier differs", position)
		}
		if walk.ReferencesByAmpersand(ctx.File.Parsed.Tokens, declParams[i]) != walk.ReferencesByAmpersand(ctx.File.Parsed.Tokens, defParams[i]) {
			return fmt.Sprintf("parameter %d reference form differs", position)
		}
		if dimensionCount(declParams[i]) != dimensionCount(defParams[i]) {
			return fmt.Sprintf("parameter %d array rank differs", position)
		}
		declDimensions := dimensions(declParams[i])
		defDimensions := dimensions(defParams[i])
		for dimension := range declDimensions {
			declValue, declKnown := ctx.Semantic.Eval(declDimensions[dimension].Field("size"))
			defValue, defKnown := ctx.Semantic.Eval(defDimensions[dimension].Field("size"))
			if declKnown && defKnown && declValue != defValue {
				return fmt.Sprintf("parameter %d array dimensions differ", position)
			}
		}
		declDefault := declParams[i].Field("default_value")
		defDefault := defParams[i].Field("default_value")
		if (declDefault == nil) != (defDefault == nil) {
			return fmt.Sprintf("parameter %d default value presence differs", position)
		}
		if declDefault != nil {
			declValue, declKnown := ctx.Semantic.Eval(declDefault)
			defValue, defKnown := ctx.Semantic.Eval(defDefault)
			if declKnown && defKnown && declValue != defValue {
				return fmt.Sprintf("parameter %d default value differs", position)
			}
		}
	}
	return ""
}

func parameters(function *parser.Node) []*parser.Node {
	list := function.Field("parameters")
	if list == nil {
		return nil
	}
	var result []*parser.Node
	for _, child := range list.Children {
		if child.Kind == parser.KindParameter {
			result = append(result, child)
		}
	}
	return result
}

func tagsKey(ctx *lint.Context, node *parser.Node) string {
	tag := node.Field("tag")
	if tag == nil {
		return ""
	}
	key := ""
	for _, child := range tag.Children {
		if child.Kind != parser.KindIdentifier {
			return ""
		}
		key += "\x00" + ctx.Walk.Text(child)
	}
	return key
}

func dimensionCount(node *parser.Node) int {
	count := 0
	for _, child := range node.Children {
		if child.Kind == parser.KindDimension {
			count++
		}
	}
	return count
}

func dimensions(node *parser.Node) []*parser.Node {
	var result []*parser.Node
	for _, child := range node.Children {
		if child.Kind == parser.KindDimension {
			result = append(result, child)
		}
	}
	return result
}

func variadic(node *parser.Node) bool {
	return node.Field("name") == nil && node.Tok.Kind == token.Ellipsis
}

func quoteFunction(name string) string {
	return "'" + name + "'"
}
