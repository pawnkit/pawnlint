package maintainability

import (
	"fmt"
	"regexp"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type TooManyParameters struct{}

func (TooManyParameters) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "too-many-parameters",
		Name:            "Too many parameters",
		Summary:         "Reports functions with too many parameters",
		Explanation:     "Named and variadic parameters count toward the configured maximum. Public functions and known callbacks are skipped by default because their signatures may be externally fixed. Name exclusions support project-specific interfaces.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"size", "functions", "parameters", "maintainability"},
		Options: []lint.Option{
			{Name: "maximum", Summary: "Maximum parameters per function", Type: lint.OptionInteger, Default: int64(7), Minimum: 1, Maximum: 1000, HasMinimum: true, HasMaximum: true},
			{Name: "include-public", Summary: "Check public function signatures", Type: lint.OptionBoolean, Default: false},
			{Name: "include-callbacks", Summary: "Check known callback signatures", Type: lint.OptionBoolean, Default: false},
			{Name: "exclude", Summary: "Function name regular expressions to exclude", Type: lint.OptionStringList, Default: []string{}, Validate: validateNamingPatterns},
		},
	}
}

type tooManyParametersOptions struct {
	maximum          int
	includePublic    bool
	includeCallbacks bool
	exclude          []*regexp.Regexp
}

func (TooManyParameters) Run(ctx *lint.Context) {
	options := configuredTooManyParameters(ctx)
	for _, function := range ctx.Walk.OfKind(parser.KindFunctionDefinition) {
		if function.HasError || ctx.Walk.Inactive(function) || ctx.Walk.Uncertain(function) {
			continue
		}
		name := function.Field("name")
		if name == nil {
			continue
		}
		nameText := ctx.Walk.Text(name)
		if !options.includePublic && walk.HasChildToken(function, token.KwPublic) || !options.includeCallbacks && callbackName(ctx, nameText) || tooManyParametersExcluded(options.exclude, nameText) {
			continue
		}
		count := definiteParameterCount(ctx, function.Field("parameters"))
		if count <= options.maximum {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("function %q has %d parameters, exceeding the maximum of %d", nameText, count, options.maximum),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(name),
		})
	}
}

func configuredTooManyParameters(ctx *lint.Context) tooManyParametersOptions {
	options := tooManyParametersOptions{maximum: 7}
	if ctx.PerRule == nil || ctx.PerRule["too-many-parameters"] == nil {
		return options
	}
	values := ctx.PerRule["too-many-parameters"]
	if value, ok := values["maximum"].(int64); ok && value > 0 {
		options.maximum = int(value)
	}
	options.includePublic, _ = values["include-public"].(bool)
	options.includeCallbacks, _ = values["include-callbacks"].(bool)
	patterns, _ := values["exclude"].([]string)
	for _, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			options.exclude = append(options.exclude, compiled)
		}
	}
	return options
}

func tooManyParametersExcluded(patterns []*regexp.Regexp, name string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

func definiteParameterCount(ctx *lint.Context, node *parser.Node) int {
	if node == nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return 0
	}
	if node.Kind == parser.KindParameter {
		return 1
	}
	count := 0
	for _, child := range node.Children {
		count += definiteParameterCount(ctx, child)
	}
	return count
}
