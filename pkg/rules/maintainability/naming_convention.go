package maintainability

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type NamingConvention struct{}

func (NamingConvention) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "naming-convention",
		Name:            "Naming convention",
		Summary:         "Reports declarations that violate configured naming policies",
		Explanation:     "Ordered conventions select symbols by kind, scope, storage, and tag. The first matching convention checks case, prefix, suffix, and an optional regular expression. Exclusion expressions suppress matching names. Callbacks and natives require explicit opt-in.",
		Category:        diagnostic.CategoryStyle,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"naming", "style", "policy", "identifiers"},
		Options: []lint.Option{{
			Name: "conventions", Summary: "Ordered naming selector and policy objects",
			Type: lint.OptionObjectList, Default: []map[string]any{}, Validate: validateNamingConventions,
			Fields: namingConventionFields(),
		}},
	}
}

func namingConventionFields() []lint.Option {
	return []lint.Option{
		{Name: "kinds", Type: lint.OptionStringList, Choices: []string{"function", "global", "local", "parameter", "enum", "enum-entry", "label"}},
		{Name: "scopes", Type: lint.OptionStringList, Choices: []string{"global", "local"}},
		{Name: "storage", Type: lint.OptionStringList, Choices: []string{"automatic", "const", "static", "public", "stock", "native", "forward", "default"}},
		{Name: "tags", Type: lint.OptionStringList},
		{Name: "case", Type: lint.OptionString, Choices: []string{"camelCase", "PascalCase", "snake_case", "UPPER_SNAKE_CASE", "lowercase", "UPPERCASE"}},
		{Name: "prefix", Type: lint.OptionString},
		{Name: "suffix", Type: lint.OptionString},
		{Name: "pattern", Type: lint.OptionString, Validate: validateNamingPattern},
		{Name: "exclude", Type: lint.OptionStringList, Validate: validateNamingPatterns},
		{Name: "include-callbacks", Type: lint.OptionBoolean, Default: false},
		{Name: "include-natives", Type: lint.OptionBoolean, Default: false},
	}
}

func validateNamingPattern(value any) error {
	pattern, _ := value.(string)
	if pattern == "" {
		return nil
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("must be a valid regular expression: %w", err)
	}
	return nil
}

func validateNamingPatterns(value any) error {
	patterns, _ := value.([]string)
	for _, pattern := range patterns {
		if err := validateNamingPattern(pattern); err != nil {
			return err
		}
	}
	return nil
}

func validateNamingConventions(value any) error {
	conventions, _ := value.([]map[string]any)
	for index, convention := range conventions {
		if namingString(convention, "case") == "" && namingString(convention, "prefix") == "" && namingString(convention, "suffix") == "" && namingString(convention, "pattern") == "" {
			return fmt.Errorf("entry %d must configure case, prefix, suffix, or pattern", index+1)
		}
	}
	return nil
}

type namingPolicy struct {
	kinds            map[string]bool
	scopes           map[string]bool
	storage          map[string]bool
	tags             map[string]bool
	caseName         string
	prefix           string
	suffix           string
	pattern          *regexp.Regexp
	exclude          []*regexp.Regexp
	includeCallbacks bool
	includeNatives   bool
}

func (NamingConvention) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	policies := configuredNamingPolicies(ctx)
	if len(policies) == 0 {
		return
	}
	definitions := make(map[string]bool)
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol.Kind == semantic.SymbolFunction && symbol.Decl != nil && symbol.Decl.Kind == parser.KindFunctionDefinition {
			definitions[symbol.Name] = true
		}
	}
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
			if !policy.matches(symbol, kind, scope, storage) || callback && !policy.includeCallbacks || native && !policy.includeNatives {
				continue
			}
			if policy.excluded(symbol.Name) {
				break
			}
			message := policy.violation(kind, symbol.Name)
			if message != "" {
				ctx.Report(diagnostic.Diagnostic{Message: message, Filename: ctx.File.Path, Range: ctx.Walk.Range(symbol.NameNode)})
			}
			break
		}
	}
}

func configuredNamingPolicies(ctx *lint.Context) []namingPolicy {
	if ctx.PerRule == nil || ctx.PerRule["naming-convention"] == nil {
		return nil
	}
	objects, _ := ctx.PerRule["naming-convention"]["conventions"].([]map[string]any)
	var result []namingPolicy
	for _, object := range objects {
		policy := namingPolicy{
			kinds:            namingSet(object, "kinds"),
			scopes:           namingSet(object, "scopes"),
			storage:          namingSet(object, "storage"),
			tags:             namingSet(object, "tags"),
			caseName:         namingString(object, "case"),
			prefix:           namingString(object, "prefix"),
			suffix:           namingString(object, "suffix"),
			includeCallbacks: namingBool(object, "include-callbacks"),
			includeNatives:   namingBool(object, "include-natives"),
		}
		if pattern := namingString(object, "pattern"); pattern != "" {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				continue
			}
			policy.pattern = compiled
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

func namingSymbol(ctx *lint.Context, symbol *semantic.Symbol, definitions map[string]bool) bool {
	if symbol == nil || symbol.Ambiguous || symbol.NameNode == nil || symbol.Decl == nil || !pawnIdentifier.MatchString(symbol.Name) {
		return false
	}
	if symbol.Kind == semantic.SymbolFunction && symbol.Decl.Kind == parser.KindFunctionDeclaration && definitions[symbol.Name] && !walk.HasChildToken(symbol.Decl, token.KwNative) {
		return false
	}
	return !ctx.Walk.Uncertain(symbol.Decl) && !ctx.Walk.Inactive(symbol.Decl)
}

func namingKind(kind semantic.SymbolKind) string {
	switch kind {
	case semantic.SymbolFunction:
		return "function"
	case semantic.SymbolGlobal:
		return "global"
	case semantic.SymbolLocal:
		return "local"
	case semantic.SymbolParameter:
		return "parameter"
	case semantic.SymbolEnumRoot:
		return "enum"
	case semantic.SymbolEnumEntry:
		return "enum-entry"
	case semantic.SymbolLabel:
		return "label"
	default:
		return ""
	}
}

func namingScope(symbol *semantic.Symbol) string {
	switch symbol.Kind {
	case semantic.SymbolLocal, semantic.SymbolParameter, semantic.SymbolLabel:
		return "local"
	default:
		return "global"
	}
}

func namingStorage(ctx *lint.Context, symbol *semantic.Symbol) []string {
	node := symbol.Decl
	if symbol.Kind == semantic.SymbolGlobal || symbol.Kind == semantic.SymbolLocal {
		node = ctx.Walk.Parent(node)
	}
	var result []string
	for _, item := range []struct {
		kind token.Kind
		name string
	}{{token.KwConst, "const"}, {token.KwStatic, "static"}, {token.KwPublic, "public"}, {token.KwStock, "stock"}, {token.KwNative, "native"}, {token.KwForward, "forward"}} {
		if walk.HasChildToken(node, item.kind) {
			result = append(result, item.name)
		}
	}
	if len(result) == 0 {
		if symbol.Kind == semantic.SymbolLocal || symbol.Kind == semantic.SymbolParameter {
			return []string{"automatic"}
		}
		return []string{"default"}
	}
	return result
}

func callbackName(ctx *lint.Context, name string) bool {
	_, ok := ctx.Callbacks()[name]
	return ok
}

func (policy namingPolicy) matches(symbol *semantic.Symbol, kind, scope string, storage []string) bool {
	if len(policy.kinds) != 0 && !policy.kinds[kind] || len(policy.scopes) != 0 && !policy.scopes[scope] {
		return false
	}
	if len(policy.storage) != 0 {
		matched := false
		for _, value := range storage {
			matched = matched || policy.storage[value]
		}
		if !matched {
			return false
		}
	}
	if len(policy.tags) == 0 {
		return true
	}
	if len(symbol.Tags) == 0 {
		return policy.tags["untagged"]
	}
	for _, tag := range symbol.Tags {
		if policy.tags[tag] {
			return true
		}
	}
	return false
}

func (policy namingPolicy) excluded(name string) bool {
	for _, pattern := range policy.exclude {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

func (policy namingPolicy) violation(kind, name string) string {
	if policy.prefix != "" && !strings.HasPrefix(name, policy.prefix) {
		return fmt.Sprintf("%s name %q must start with %q", kind, name, policy.prefix)
	}
	if policy.suffix != "" && !strings.HasSuffix(name, policy.suffix) {
		return fmt.Sprintf("%s name %q must end with %q", kind, name, policy.suffix)
	}
	core := strings.TrimPrefix(name, policy.prefix)
	core = strings.TrimSuffix(core, policy.suffix)
	if policy.caseName != "" && !namingCase(policy.caseName, core) {
		return fmt.Sprintf("%s name %q must use %s", kind, name, policy.caseName)
	}
	if policy.pattern != nil && !policy.pattern.MatchString(name) {
		return fmt.Sprintf("%s name %q must match pattern %q", kind, name, policy.pattern.String())
	}
	return ""
}

func namingCase(name, value string) bool {
	pattern := namingCases[name]
	return pattern != nil && pattern.MatchString(value)
}

func namingSet(object map[string]any, name string) map[string]bool {
	values := namingStrings(object, name)
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func namingStrings(object map[string]any, name string) []string {
	values, _ := object[name].([]string)
	return values
}

func namingString(object map[string]any, name string) string {
	value, _ := object[name].(string)
	return value
}

func namingBool(object map[string]any, name string) bool {
	value, _ := object[name].(bool)
	return value
}

func containsNamingValue(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

var pawnIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var namingCases = map[string]*regexp.Regexp{
	"camelCase":        regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`),
	"PascalCase":       regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`),
	"snake_case":       regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`),
	"UPPER_SNAKE_CASE": regexp.MustCompile(`^[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*$`),
	"lowercase":        regexp.MustCompile(`^[a-z][a-z0-9]*$`),
	"UPPERCASE":        regexp.MustCompile(`^[A-Z][A-Z0-9]*$`),
}
