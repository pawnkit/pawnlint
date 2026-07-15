package maintainability

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DisallowedName struct{}

func (DisallowedName) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "disallowed-name",
		Name:            "Disallowed name",
		Summary:         "Reports declarations denied by configured name policies",
		Explanation:     "Configured policies deny exact names or regular-expression matches for selected symbol kinds, scopes, storage classes, and tags. Exclusions override a policy. Callbacks and natives require explicit opt-in.",
		Category:        diagnostic.CategoryRestriction,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"naming", "restriction", "policy", "identifiers"},
		Options: []lint.Option{{
			Name: "policies", Summary: "Name deny-list policy objects",
			Type: lint.OptionObjectList, Default: []map[string]any{}, Validate: validateDisallowedPolicies,
			Fields: disallowedNameFields(),
		}},
		ConfigExample: `[rules.disallowed-name]
severity = "warning"
policies = [
  { kinds = ["local", "parameter"], names = ["foo", "bar"] },
  { patterns = ["^temp_"], exclude = ["^temporaryAllowed$"] }
]`,
	}
}

func disallowedNameFields() []lint.Option {
	fields := namingSelectorFields()
	return append(fields,
		lint.Option{Name: "names", Summary: "Exact names this policy denies", Type: lint.OptionStringList, Validate: validateExactNames},
		lint.Option{Name: "patterns", Summary: "Regular expressions this policy denies", Type: lint.OptionStringList, Validate: validateDisallowedPatterns},
		lint.Option{Name: "exclude", Summary: "Regular expressions that exempt a matching name", Type: lint.OptionStringList, Validate: validateNamingPatterns},
		lint.Option{Name: "reason", Summary: "Message appended to the diagnostic explaining the policy", Type: lint.OptionString},
	)
}

func validateExactNames(value any) error {
	names, _ := value.([]string)
	for _, name := range names {
		if strings.TrimSpace(name) != name || !pawnIdentifier.MatchString(name) {
			return fmt.Errorf("entries must be supported Pawn identifiers")
		}
	}
	return nil
}

func validateDisallowedPatterns(value any) error {
	patterns, _ := value.([]string)
	for _, pattern := range patterns {
		if pattern == "" {
			return fmt.Errorf("entries must be non-empty")
		}
		if err := validateNamingPattern(pattern); err != nil {
			return err
		}
	}
	return nil
}

func validateDisallowedPolicies(value any) error {
	policies, _ := value.([]map[string]any)
	for index, policy := range policies {
		if len(namingStrings(policy, "names")) == 0 && len(namingStrings(policy, "patterns")) == 0 {
			return fmt.Errorf("entry %d must configure names or patterns", index+1)
		}
	}
	return nil
}

type deniedNamePolicy struct {
	selector namingPolicy
	names    map[string]bool
	patterns []*regexp.Regexp
	exclude  []*regexp.Regexp
	reason   string
}

func (DisallowedName) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	policies := configuredDeniedNamePolicies(ctx)
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
			if !selector.matches(symbol, kind, scope, storage) || callback && !selector.includeCallbacks || native && !selector.includeNatives || deniedNameExcluded(policy, symbol.Name) {
				continue
			}
			pattern, denied := deniedNameMatch(policy, symbol.Name)
			if !denied {
				continue
			}
			message := fmt.Sprintf("%s name %q is disallowed", kind, symbol.Name)
			if pattern != "" {
				message += fmt.Sprintf(" by pattern %q", pattern)
			}
			if policy.reason != "" {
				message += ": " + policy.reason
			}
			ctx.Report(diagnostic.Diagnostic{Message: message, Filename: ctx.File.Path, Range: ctx.Walk.Range(symbol.NameNode)})
			break
		}
	}
}

func configuredDeniedNamePolicies(ctx *lint.Context) []deniedNamePolicy {
	if ctx.PerRule == nil || ctx.PerRule["disallowed-name"] == nil {
		return nil
	}
	objects, _ := ctx.PerRule["disallowed-name"]["policies"].([]map[string]any)
	var result []deniedNamePolicy
	for _, object := range objects {
		policy := deniedNamePolicy{
			selector: namingPolicy{
				kinds:            namingSet(object, "kinds"),
				scopes:           namingSet(object, "scopes"),
				storage:          namingSet(object, "storage"),
				tags:             namingSet(object, "tags"),
				includeCallbacks: namingBool(object, "include-callbacks"),
				includeNatives:   namingBool(object, "include-natives"),
			},
			names:  namingSet(object, "names"),
			reason: namingString(object, "reason"),
		}
		valid := true
		for _, pattern := range namingStrings(object, "patterns") {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				valid = false
				break
			}
			policy.patterns = append(policy.patterns, compiled)
		}
		if !valid {
			continue
		}
		for _, pattern := range namingStrings(object, "exclude") {
			compiled, err := regexp.Compile(pattern)
			if err == nil {
				policy.exclude = append(policy.exclude, compiled)
			}
		}
		result = append(result, policy)
	}
	return result
}

func deniedNameExcluded(policy deniedNamePolicy, name string) bool {
	for _, pattern := range policy.exclude {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

func deniedNameMatch(policy deniedNamePolicy, name string) (string, bool) {
	if policy.names[name] {
		return "", true
	}
	for _, pattern := range policy.patterns {
		if pattern.MatchString(name) {
			return pattern.String(), true
		}
	}
	return "", false
}
