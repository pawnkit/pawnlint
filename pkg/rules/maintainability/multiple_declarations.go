package maintainability

import (
	"fmt"
	"sort"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type MultipleDeclarations struct{}

var multipleDeclarationScopes = []string{"global", "local"}

func (MultipleDeclarations) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "multiple-declarations",
		Name:            "Multiple declarations",
		Summary:         "Reports statements that declare multiple variables",
		Explanation:     "Configured global and local declarations must contain one variable declarator. Multi-variable for-loop initializers can be allowed separately. Inactive, uncertain, and malformed declarations are ignored.",
		Category:        diagnostic.CategoryStyle,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"declarations", "style", "variables"},
		Options: []lint.Option{
			{Name: "scopes", Summary: "Declaration scopes to check", Type: lint.OptionStringList, Default: multipleDeclarationScopes, Choices: []string{"global", "local"}, Validate: validateMultipleDeclarationScopes},
			{Name: "allow-for-loop", Summary: "Allow multiple variables in for-loop initializers", Type: lint.OptionBoolean, Default: true},
		},
	}
}

func validateMultipleDeclarationScopes(value any) error {
	values, _ := value.([]string)
	if len(values) == 0 {
		return fmt.Errorf("must contain at least one entry")
	}
	return nil
}

type multipleDeclarationsOptions struct {
	scopes       map[string]bool
	allowForLoop bool
}

func (MultipleDeclarations) Run(ctx *lint.Context) {
	options := configuredMultipleDeclarations(ctx)
	for _, declaration := range ctx.Walk.OfKind(parser.KindVariableDeclaration) {
		if declaration.HasError || ctx.Walk.Inactive(declaration) || ctx.Walk.Uncertain(declaration) {
			continue
		}
		scope := "global"
		if ctx.Walk.EnclosingFunction(declaration) != nil {
			scope = "local"
		}
		if !options.scopes[scope] || options.allowForLoop && forLoopDeclaration(ctx, declaration) {
			continue
		}
		declarators := definiteVariableDeclarators(ctx, declaration)
		if len(declarators) < 2 {
			continue
		}
		name := declarators[1].Field("name")
		if name == nil {
			name = declarators[1]
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("declare one variable per statement; this declaration contains %d variables", len(declarators)),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(name),
		})
	}
}

func configuredMultipleDeclarations(ctx *lint.Context) multipleDeclarationsOptions {
	options := multipleDeclarationsOptions{scopes: map[string]bool{"global": true, "local": true}, allowForLoop: true}
	if ctx.PerRule == nil || ctx.PerRule["multiple-declarations"] == nil {
		return options
	}
	values := ctx.PerRule["multiple-declarations"]
	if scopes, ok := values["scopes"].([]string); ok {
		options.scopes = make(map[string]bool, len(scopes))
		for _, scope := range scopes {
			options.scopes[scope] = true
		}
	}
	if value, ok := values["allow-for-loop"].(bool); ok {
		options.allowForLoop = value
	}
	return options
}

func forLoopDeclaration(ctx *lint.Context, declaration *parser.Node) bool {
	parent := ctx.Walk.Parent(declaration)
	return parent != nil && parent.Kind == parser.KindForStatement && parent.Field("init") == declaration
}

func definiteVariableDeclarators(ctx *lint.Context, root *parser.Node) []*parser.Node {
	var result []*parser.Node
	var visit func(*parser.Node)
	visit = func(node *parser.Node) {
		if node == nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
			return
		}
		if node.Kind == parser.KindVariableDeclarator {
			result = append(result, node)
			return
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(root)
	sort.SliceStable(result, func(left, right int) bool { return result[left].Start < result[right].Start })
	return result
}
