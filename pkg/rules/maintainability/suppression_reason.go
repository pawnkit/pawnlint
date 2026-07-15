package maintainability

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/lint/suppress"
)

type SuppressionReason struct{}

func (SuppressionReason) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "suppression-reason",
		Name:            "Suppression reason",
		Summary:         "Reports suppression directives without an adequate reason",
		Explanation:     "Disable directives must include a reason after --. A configurable minimum length prevents empty explanations, and an optional regular expression can require issue or ticket formats. Enable and malformed directives are handled separately.",
		Category:        diagnostic.CategoryRestriction,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"suppression", "policy", "documentation"},
		Options: []lint.Option{
			{Name: "minimum-length", Summary: "Minimum number of characters in a reason", Type: lint.OptionInteger, Default: int64(1), Minimum: 1, Maximum: 1000, HasMinimum: true, HasMaximum: true},
			{Name: "pattern", Summary: "Regular expression required in each reason", Type: lint.OptionString, Default: "", Validate: validateSuppressionReasonPattern},
		},
	}
}

func validateSuppressionReasonPattern(value any) error {
	pattern, _ := value.(string)
	if pattern == "" {
		return nil
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("must be a valid regular expression: %w", err)
	}
	return nil
}

func (SuppressionReason) Run(ctx *lint.Context) {
	minimum, pattern := suppressionReasonOptions(ctx)
	for _, directive := range ctx.Supp {
		if directive.Kind == suppress.KindEnable || directive.Malformed {
			continue
		}
		message := ""
		switch {
		case directive.Reason == "":
			message = "suppression must include a reason after --"
		case utf8.RuneCountInString(directive.Reason) < minimum:
			message = fmt.Sprintf("suppression reason must contain at least %d characters", minimum)
		case pattern != nil && !pattern.MatchString(directive.Reason):
			message = fmt.Sprintf("suppression reason must match pattern %q", pattern.String())
		}
		if message == "" {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message: message,
			Range:   ctx.Walk.LineTable.Range(directive.Offset, directive.End),
		})
	}
}

func suppressionReasonOptions(ctx *lint.Context) (int, *regexp.Regexp) {
	minimum := 1
	if ctx.PerRule == nil || ctx.PerRule["suppression-reason"] == nil {
		return minimum, nil
	}
	values := ctx.PerRule["suppression-reason"]
	if value, ok := values["minimum-length"].(int64); ok {
		minimum = int(value)
	}
	pattern, _ := values["pattern"].(string)
	if pattern == "" {
		return minimum, nil
	}
	compiled, _ := regexp.Compile(pattern)
	return minimum, compiled
}
