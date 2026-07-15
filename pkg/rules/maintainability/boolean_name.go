package maintainability

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type BooleanName struct{}

func (BooleanName) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "boolean-name",
		Name:            "Boolean name",
		Summary:         "Reports boolean declarations without an allowed prefix",
		Explanation:     "Ordered policies select declarations with a definite bool tag. The first matching policy requires one configured prefix at a naming boundary. Exclusions override a policy. Callbacks and natives require explicit opt-in.",
		Category:        diagnostic.CategoryStyle,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"naming", "style", "policy", "boolean"},
		Options: []lint.Option{{
			Name: "policies", Summary: "Ordered boolean-name selector and prefix objects",
			Type: lint.OptionObjectList, Default: []map[string]any{}, Validate: validateBooleanNamePolicies,
			Fields: booleanNameFields(),
		}},
		ConfigExample: `[rules.boolean-name]
severity = "warning"
policies = [
  { kinds = ["function"], prefixes = ["Is", "Has", "Can"] },
  { kinds = ["global", "local", "parameter"], prefixes = ["is", "has", "can", "b_"] }
]`,
	}
}

func booleanNameFields() []lint.Option {
	fields := namingSelectorFields()
	fields[0].Choices = []string{"function", "global", "local", "parameter"}
	return append(fields,
		lint.Option{Name: "prefixes", Summary: "Allowed prefixes; the name must start with one at a naming boundary", Type: lint.OptionStringList, Validate: validateBooleanPrefixes},
		lint.Option{Name: "exclude", Summary: "Regular expressions that exempt a matching name", Type: lint.OptionStringList, Validate: validateNamingPatterns},
	)
}

func validateBooleanPrefixes(value any) error {
	prefixes, _ := value.([]string)
	for _, prefix := range prefixes {
		if !pawnIdentifier.MatchString(prefix) {
			return fmt.Errorf("entries must be supported Pawn identifier prefixes")
		}
	}
	return nil
}

func validateBooleanNamePolicies(value any) error {
	policies, _ := value.([]map[string]any)
	for index, policy := range policies {
		if len(namingStrings(policy, "prefixes")) == 0 {
			return fmt.Errorf("entry %d must configure prefixes", index+1)
		}
	}
	return nil
}

type booleanNamePolicy struct {
	selector namingPolicy
	prefixes []string
	exclude  []*regexp.Regexp
}

func (BooleanName) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	policies := configuredBooleanNamePolicies(ctx)
	if len(policies) == 0 {
		return
	}
	definitions := namingDefinitions(ctx)
	for _, symbol := range ctx.Semantic.Symbols {
		if !namingSymbol(ctx, symbol, definitions) || !definiteBooleanSymbol(symbol) {
			continue
		}
		kind := namingKind(symbol.Kind)
		if kind != "function" && kind != "global" && kind != "local" && kind != "parameter" {
			continue
		}
		scope := namingScope(symbol)
		storage := namingStorage(ctx, symbol)
		callback := kind == "function" && containsNamingValue(storage, "public") && callbackName(ctx, symbol.Name)
		native := kind == "function" && containsNamingValue(storage, "native")
		for _, policy := range policies {
			selector := policy.selector
			if !selector.matches(symbol, kind, scope, storage) || callback && !selector.includeCallbacks || native && !selector.includeNatives {
				continue
			}
			if booleanNameExcluded(policy, symbol.Name) {
				break
			}
			if !matchesBooleanPrefix(symbol.Name, policy.prefixes) {
				ctx.Report(diagnostic.Diagnostic{
					Message:  fmt.Sprintf("boolean %s name %q must start with %s", kind, symbol.Name, booleanPrefixList(policy.prefixes)),
					Filename: ctx.File.Path,
					Range:    ctx.Walk.Range(symbol.NameNode),
				})
			}
			break
		}
	}
}

func configuredBooleanNamePolicies(ctx *lint.Context) []booleanNamePolicy {
	if ctx.PerRule == nil || ctx.PerRule["boolean-name"] == nil {
		return nil
	}
	objects, _ := ctx.PerRule["boolean-name"]["policies"].([]map[string]any)
	var result []booleanNamePolicy
	for _, object := range objects {
		policy := booleanNamePolicy{
			selector: namingPolicy{
				kinds:            namingSet(object, "kinds"),
				scopes:           namingSet(object, "scopes"),
				storage:          namingSet(object, "storage"),
				tags:             namingSet(object, "tags"),
				includeCallbacks: namingBool(object, "include-callbacks"),
				includeNatives:   namingBool(object, "include-natives"),
			},
			prefixes: namingStrings(object, "prefixes"),
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

func definiteBooleanSymbol(symbol *semantic.Symbol) bool {
	return len(symbol.Tags) == 1 && symbol.Tags[0] == "bool"
}

func booleanNameExcluded(policy booleanNamePolicy, name string) bool {
	for _, pattern := range policy.exclude {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

func matchesBooleanPrefix(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if !strings.HasPrefix(name, prefix) || len(name) == len(prefix) {
			continue
		}
		if prefix[len(prefix)-1] == '_' || booleanNameBoundary(name[len(prefix)]) {
			return true
		}
	}
	return false
}

func booleanNameBoundary(value byte) bool {
	return value == '_' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

func booleanPrefixList(prefixes []string) string {
	quoted := make([]string, len(prefixes))
	for index, prefix := range prefixes {
		quoted[index] = fmt.Sprintf("%q", prefix)
	}
	switch len(quoted) {
	case 1:
		return quoted[0]
	case 2:
		return quoted[0] + " or " + quoted[1]
	default:
		return strings.Join(quoted[:len(quoted)-1], ", ") + ", or " + quoted[len(quoted)-1]
	}
}
