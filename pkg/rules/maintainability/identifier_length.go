package maintainability

import (
	"fmt"
	"regexp"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type IdentifierLength struct{}

func (IdentifierLength) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "identifier-length",
		Name:            "Identifier length",
		Summary:         "Reports declarations outside configured name-length limits",
		Explanation:     "Ordered limits select declarations by kind, scope, storage, and tag. The first matching limit checks minimum and maximum ASCII identifier lengths. Callbacks and natives require explicit opt-in. One-character loop indices can be allowed when their role is definite.",
		Category:        diagnostic.CategoryStyle,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"naming", "style", "policy", "identifiers"},
		Options: []lint.Option{{
			Name: "limits", Summary: "Ordered identifier-length selector and limit objects",
			Type: lint.OptionObjectList, Default: []map[string]any{}, Validate: validateIdentifierLimits,
			Fields: identifierLengthFields(),
		}},
		ConfigExample: `[rules.identifier-length]
severity = "warning"
limits = [
  { kinds = ["function", "global"], minimum = 3, maximum = 40 },
  { kinds = ["local", "parameter"], minimum = 2, maximum = 30, exclude = ["^[xyz]$"] }
]`,
	}
}

func identifierLengthFields() []lint.Option {
	fields := namingSelectorFields()
	return append(fields,
		lint.Option{Name: "minimum", Summary: "Minimum allowed identifier length", Type: lint.OptionInteger, Minimum: 1, Maximum: 1024, HasMinimum: true, HasMaximum: true},
		lint.Option{Name: "maximum", Summary: "Maximum allowed identifier length", Type: lint.OptionInteger, Minimum: 1, Maximum: 1024, HasMinimum: true, HasMaximum: true},
		lint.Option{Name: "exclude", Summary: "Regular expressions that exempt a matching name", Type: lint.OptionStringList, Validate: validateNamingPatterns},
		lint.Option{Name: "allow-loop-indices", Summary: "Allow single-character for-loop indices", Type: lint.OptionBoolean, Default: true},
	)
}

func validateIdentifierLimits(value any) error {
	limits, _ := value.([]map[string]any)
	for index, limit := range limits {
		minimum, hasMinimum := namingInteger(limit, "minimum")
		maximum, hasMaximum := namingInteger(limit, "maximum")
		if !hasMinimum && !hasMaximum {
			return fmt.Errorf("entry %d must configure minimum or maximum", index+1)
		}
		if hasMinimum && hasMaximum && minimum > maximum {
			return fmt.Errorf("entry %d minimum must not exceed maximum", index+1)
		}
	}
	return nil
}

type identifierLengthPolicy struct {
	selector         namingPolicy
	minimum          int
	maximum          int
	hasMinimum       bool
	hasMaximum       bool
	exclude          []*regexp.Regexp
	allowLoopIndices bool
}

func (IdentifierLength) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	policies := configuredIdentifierLimits(ctx)
	if len(policies) == 0 {
		return
	}
	definitions := namingDefinitions(ctx)
	for _, symbol := range ctx.Semantic.Symbols {
		if !namingSymbol(ctx, symbol, definitions) {
			continue
		}
		kind := namingKind(symbol.Kind)
		scope := namingScope(symbol)
		storage := namingStorage(ctx, symbol)
		callback := kind == "function" && containsNamingValue(storage, "public") && callbackName(ctx, symbol.Name)
		native := kind == "function" && containsNamingValue(storage, "native")
		for _, policy := range policies {
			selector := policy.selector
			if !selector.matches(symbol, kind, scope, storage) || callback && !selector.includeCallbacks || native && !selector.includeNatives {
				continue
			}
			if identifierLengthExcluded(policy, symbol.Name) || policy.allowLoopIndices && definiteLoopIndex(ctx, symbol) {
				break
			}
			length := len(symbol.Name)
			message := ""
			if policy.hasMinimum && length < policy.minimum {
				message = fmt.Sprintf("%s name %q is shorter than the minimum of %d characters", kind, symbol.Name, policy.minimum)
			} else if policy.hasMaximum && length > policy.maximum {
				message = fmt.Sprintf("%s name %q is longer than the maximum of %d characters", kind, symbol.Name, policy.maximum)
			}
			if message != "" {
				ctx.Report(diagnostic.Diagnostic{Message: message, Filename: ctx.File.Path, Range: ctx.Walk.Range(symbol.NameNode)})
			}
			break
		}
	}
}

func configuredIdentifierLimits(ctx *lint.Context) []identifierLengthPolicy {
	if ctx.PerRule == nil || ctx.PerRule["identifier-length"] == nil {
		return nil
	}
	objects, _ := ctx.PerRule["identifier-length"]["limits"].([]map[string]any)
	var result []identifierLengthPolicy
	for _, object := range objects {
		minimum, hasMinimum := namingInteger(object, "minimum")
		maximum, hasMaximum := namingInteger(object, "maximum")
		policy := identifierLengthPolicy{
			selector: namingPolicy{
				kinds:            namingSet(object, "kinds"),
				scopes:           namingSet(object, "scopes"),
				storage:          namingSet(object, "storage"),
				tags:             namingSet(object, "tags"),
				includeCallbacks: namingBool(object, "include-callbacks"),
				includeNatives:   namingBool(object, "include-natives"),
			},
			minimum:          int(minimum),
			maximum:          int(maximum),
			hasMinimum:       hasMinimum,
			hasMaximum:       hasMaximum,
			allowLoopIndices: namingBool(object, "allow-loop-indices"),
		}
		valid := true
		for _, pattern := range namingStrings(object, "exclude") {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				valid = false
				break
			}
			policy.exclude = append(policy.exclude, compiled)
		}
		if valid {
			result = append(result, policy)
		}
	}
	return result
}

func namingInteger(object map[string]any, name string) (int64, bool) {
	value, ok := object[name].(int64)
	return value, ok
}

func identifierLengthExcluded(policy identifierLengthPolicy, name string) bool {
	for _, pattern := range policy.exclude {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

func definiteLoopIndex(ctx *lint.Context, symbol *semantic.Symbol) bool {
	if symbol.Kind != semantic.SymbolLocal || len(symbol.Name) != 1 {
		return false
	}
	declaration := ctx.Walk.Parent(symbol.Decl)
	loop := ctx.Walk.Parent(declaration)
	if declaration == nil || loop == nil || loop.Kind != parser.KindForStatement || loop.Field("init") != declaration {
		return false
	}
	condition := loop.Field("condition")
	increment := loop.Field("increment")
	for _, reference := range ctx.Semantic.References(symbol) {
		if identifierInside(ctx, reference.Node, condition) || identifierInside(ctx, reference.Node, increment) {
			return true
		}
	}
	return false
}

func identifierInside(ctx *lint.Context, node, owner *parser.Node) bool {
	if owner == nil {
		return false
	}
	if node == owner {
		return true
	}
	for ancestor := ctx.Walk.Parent(node); ancestor != nil; ancestor = ctx.Walk.Parent(ancestor) {
		if ancestor == owner {
			return true
		}
	}
	return false
}
