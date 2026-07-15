package maintainability

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type MagicValue struct{}

func (MagicValue) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "magic-value",
		Name:            "Magic value",
		Summary:         "Reports unexplained numeric and string literals",
		Explanation:     "Magic values hide the meaning of policy and domain constants. Named constants make reuse and changes safer. Declaration, array, and function-call exemptions keep generated data and fixed interfaces out of scope.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"constants", "literals", "maintainability", "policy"},
		Options: []lint.Option{
			{Name: "check-numbers", Summary: "Check numeric literals", Type: lint.OptionBoolean, Default: true},
			{Name: "check-strings", Summary: "Check string literals", Type: lint.OptionBoolean, Default: false},
			{Name: "allowed-numbers", Summary: "Numeric values to allow", Type: lint.OptionStringList, Default: []string{"-1", "0", "1", "-1.0", "0.0", "1.0"}, Validate: validateMagicNumbers},
			{Name: "allowed-strings", Summary: "String contents to allow", Type: lint.OptionStringList, Default: []string{}},
			{Name: "ignore-const-declarations", Summary: "Ignore literals in const declarations", Type: lint.OptionBoolean, Default: true},
			{Name: "ignore-enums", Summary: "Ignore literals in enum declarations", Type: lint.OptionBoolean, Default: true},
			{Name: "ignore-default-parameters", Summary: "Ignore parameter default values", Type: lint.OptionBoolean, Default: true},
			{Name: "ignore-array-sizes", Summary: "Ignore array dimensions", Type: lint.OptionBoolean, Default: true},
			{Name: "ignore-array-indexes", Summary: "Ignore array indexes", Type: lint.OptionBoolean, Default: true},
			{Name: "ignore-array-literals", Summary: "Ignore values in array literals", Type: lint.OptionBoolean, Default: true},
			{Name: "ignore-functions", Summary: "Function name regular expressions whose arguments are ignored", Type: lint.OptionStringList, Default: []string{}, Validate: validateNamingPatterns},
		},
	}
}

type magicValueOptions struct {
	checkNumbers            bool
	checkStrings            bool
	allowedNumbers          map[string]struct{}
	allowedStrings          map[string]struct{}
	ignoreConstDeclarations bool
	ignoreEnums             bool
	ignoreDefaultParameters bool
	ignoreArraySizes        bool
	ignoreArrayIndexes      bool
	ignoreArrayLiterals     bool
	ignoreFunctions         []*regexp.Regexp
}

func (MagicValue) Run(ctx *lint.Context) {
	options := configuredMagicValueOptions(ctx)
	for _, literal := range ctx.Walk.OfKind(parser.KindLiteral) {
		if literal.HasError || ctx.Walk.Inactive(literal) || ctx.Walk.Uncertain(literal) || literal.Tok.Origin != nil {
			continue
		}
		candidate := magicValueCandidate(ctx, literal)
		if magicValueIgnored(ctx, candidate, options) {
			continue
		}
		switch literal.Tok.Kind {
		case token.IntLiteral, token.FloatLiteral:
			if !options.checkNumbers {
				continue
			}
			text := ctx.Walk.Text(candidate)
			key, ok := magicNumberKey(text)
			if !ok {
				continue
			}
			if _, allowed := options.allowedNumbers[key]; allowed {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{Message: fmt.Sprintf("numeric literal %s is a magic value", text), Filename: ctx.File.Path, Range: ctx.Walk.Range(candidate)})
		case token.StringLiteral, token.PackedString:
			if !options.checkStrings {
				continue
			}
			if _, allowed := options.allowedStrings[magicStringValue(ctx.Walk.Text(literal))]; allowed {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{Message: "string literal is a magic value", Filename: ctx.File.Path, Range: ctx.Walk.Range(literal)})
		}
	}
}

func configuredMagicValueOptions(ctx *lint.Context) magicValueOptions {
	options := magicValueOptions{
		checkNumbers:            true,
		allowedNumbers:          make(map[string]struct{}),
		allowedStrings:          make(map[string]struct{}),
		ignoreConstDeclarations: true,
		ignoreEnums:             true,
		ignoreDefaultParameters: true,
		ignoreArraySizes:        true,
		ignoreArrayIndexes:      true,
		ignoreArrayLiterals:     true,
	}
	values := map[string]any(nil)
	if ctx.PerRule != nil {
		values = ctx.PerRule["magic-value"]
	}
	if value, ok := values["check-numbers"].(bool); ok {
		options.checkNumbers = value
	}
	if value, ok := values["check-strings"].(bool); ok {
		options.checkStrings = value
	}
	allowedNumbers := []string{"-1", "0", "1", "-1.0", "0.0", "1.0"}
	if value, ok := values["allowed-numbers"].([]string); ok {
		allowedNumbers = value
	}
	for _, value := range allowedNumbers {
		if key, ok := magicNumberKey(value); ok {
			options.allowedNumbers[key] = struct{}{}
		}
	}
	if allowed, ok := values["allowed-strings"].([]string); ok {
		for _, value := range allowed {
			options.allowedStrings[value] = struct{}{}
		}
	}
	magicBooleanOption(values, "ignore-const-declarations", &options.ignoreConstDeclarations)
	magicBooleanOption(values, "ignore-enums", &options.ignoreEnums)
	magicBooleanOption(values, "ignore-default-parameters", &options.ignoreDefaultParameters)
	magicBooleanOption(values, "ignore-array-sizes", &options.ignoreArraySizes)
	magicBooleanOption(values, "ignore-array-indexes", &options.ignoreArrayIndexes)
	magicBooleanOption(values, "ignore-array-literals", &options.ignoreArrayLiterals)
	patterns, _ := values["ignore-functions"].([]string)
	for _, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			options.ignoreFunctions = append(options.ignoreFunctions, compiled)
		}
	}
	return options
}

func magicBooleanOption(values map[string]any, name string, target *bool) {
	if value, ok := values[name].(bool); ok {
		*target = value
	}
}

func validateMagicNumbers(value any) error {
	values, _ := value.([]string)
	for _, item := range values {
		if _, ok := magicNumberKey(item); !ok {
			return fmt.Errorf("%q is not a numeric literal", item)
		}
	}
	return nil
}

func magicNumberKey(text string) (string, bool) {
	text = strings.ReplaceAll(strings.TrimSpace(text), "_", "")
	if strings.ContainsAny(text, ".eE") {
		value, err := strconv.ParseFloat(text, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return "", false
		}
		if value == 0 {
			value = 0
		}
		return "f:" + strconv.FormatFloat(value, 'g', -1, 64), true
	}
	if value, err := strconv.ParseInt(text, 0, 32); err == nil {
		return "i:" + strconv.FormatInt(value, 10), true
	}
	sign := int64(1)
	unsigned := text
	if strings.HasPrefix(unsigned, "+") {
		unsigned = unsigned[1:]
	} else if strings.HasPrefix(unsigned, "-") {
		sign = -1
		unsigned = unsigned[1:]
	}
	if !strings.HasPrefix(unsigned, "0x") && !strings.HasPrefix(unsigned, "0X") && !strings.HasPrefix(unsigned, "0b") && !strings.HasPrefix(unsigned, "0B") {
		return "", false
	}
	value, err := strconv.ParseUint(unsigned, 0, 32)
	if err != nil {
		return "", false
	}
	return "i:" + strconv.FormatInt(sign*int64(int32(uint32(value))), 10), true
}

func magicValueCandidate(ctx *lint.Context, literal *parser.Node) *parser.Node {
	parent := ctx.Walk.Parent(literal)
	if parent != nil && parent.Kind == parser.KindUnaryExpression && parent.Field("expression") == literal && (parent.Tok.Kind == token.Plus || parent.Tok.Kind == token.Minus) {
		return parent
	}
	return literal
}

func magicValueIgnored(ctx *lint.Context, candidate *parser.Node, options magicValueOptions) bool {
	for current := candidate; current != nil; current = ctx.Walk.Parent(current) {
		if strings.HasPrefix(current.Kind.String(), "directive_") {
			return true
		}
		switch current.Kind {
		case parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock, parser.KindSharedConditional, parser.KindConditionalFunction, parser.KindConditionalSplice:
			return true
		case parser.KindEnumDeclaration, parser.KindEnumEntry:
			if options.ignoreEnums {
				return true
			}
		case parser.KindDimension:
			if options.ignoreArraySizes {
				return true
			}
		case parser.KindParameter:
			if options.ignoreDefaultParameters && magicValueContains(current.Field("default_value"), candidate) {
				return true
			}
		case parser.KindSubscriptExpression:
			if options.ignoreArrayIndexes && magicValueContains(current.Field("index"), candidate) {
				return true
			}
		case parser.KindArrayLiteral:
			if options.ignoreArrayLiterals {
				return true
			}
		case parser.KindCallExpression:
			if magicValueContains(current.Field("arguments"), candidate) && magicValueIgnoredFunction(ctx.Walk.Text(current.Field("function")), options.ignoreFunctions) {
				return true
			}
		case parser.KindVariableDeclaration:
			if options.ignoreConstDeclarations && walk.HasChildToken(current, token.KwConst) {
				return true
			}
		}
	}
	return false
}

func magicValueIgnoredFunction(name string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

func magicValueContains(root, target *parser.Node) bool {
	if root == nil || target == nil {
		return false
	}
	return target.Start >= root.Start && target.End <= root.End
}

func magicStringValue(text string) string {
	text = strings.TrimPrefix(text, "!")
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1]
	}
	return text
}
