package lint

import (
	"strings"

	analysis "github.com/pawnkit/pawn-analysis"
	corediagnostic "github.com/pawnkit/pawnkit-core/diagnostic"
	coresource "github.com/pawnkit/pawnkit-core/source"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func appendSharedDiagnostics(dst []diagnostic.Diagnostic, path string, content []byte, shared *analysis.Result) []diagnostic.Diagnostic {
	result := shared
	if result == nil {
		result = analysis.Analyze(content, analysis.Options{URI: coresource.FileURI(path)})
	}
	for _, item := range result.Diagnostics {
		if !sharedDiagnostic(item.Code) ||
			item.Primary.File != result.File && !sharedProjectDiagnostic(item.Code) {
			continue
		}
		filename, itemContent := sharedFile(result, item.Primary.File, path, content)
		lines := source.NewLineTable(itemContent)
		start, end := int(item.Primary.Start), int(item.Primary.End)
		if duplicateShared(dst, item.Code, filename, start, end) {
			continue
		}
		ruleID := item.Code
		if equivalent := sharedRuleID(item.Code); shared != nil && equivalent != "" {
			ruleID = equivalent
		}
		dst = append(dst, diagnostic.Diagnostic{
			RuleID: ruleID, Code: item.Code, Severity: sharedSeverity(item.Severity),
			Category: diagnostic.CategoryCorrectness, Message: item.Message,
			Filename: filename, Range: lines.Range(start, end),
		})
	}
	return dst
}

func duplicateShared(dst []diagnostic.Diagnostic, code, filename string, start, end int) bool {
	equivalent := sharedRuleID(code)
	if equivalent == "" {
		return false
	}
	for _, item := range dst {
		if item.RuleID == equivalent && item.Filename == filename &&
			rangesOverlap(start, end, item.Range.Start.Offset, item.Range.End.Offset) {
			return true
		}
	}
	return false
}

func sharedRuleID(code string) string {
	return map[string]string{
		"pawn-analysis:sema/not-callable":            "non-callable-symbol",
		"pawn-analysis:sema/tag-mismatch":            "argument-tag-mismatch",
		"pawn-analysis:sema/unreachable":             "unreachable-code",
		"pawn-analysis:sema/missing-return":          "missing-return-value",
		"pawn-analysis:preprocess/include-not-found": "missing-include",
		"pawn-analysis:preprocess/include-cycle":     "include-cycle",
	}[code]
}

// DelegatesToShared reports rules owned by pawn-analysis in editor runs.
func DelegatesToShared(ruleID string) bool {
	switch ruleID {
	case "non-callable-symbol", "argument-tag-mismatch", "unreachable-code", "missing-return-value",
		"missing-include", "include-cycle":
		return true
	default:
		return false
	}
}

func sharedDiagnostic(code string) bool {
	return strings.HasPrefix(code, "pawn-analysis:sema/") || sharedRuleID(code) != ""
}

func sharedProjectDiagnostic(code string) bool {
	return code == "pawn-analysis:preprocess/include-not-found" ||
		code == "pawn-analysis:preprocess/include-cycle"
}

func sharedFile(result *analysis.Result, file coresource.FileID, fallback string, content []byte) (string, []byte) {
	if result == nil || result.Registry == nil {
		return fallback, content
	}
	uri, ok := result.Registry.URI(file)
	if !ok {
		return fallback, content
	}
	filename, err := uri.Filename()
	if err != nil {
		return fallback, content
	}
	if result.Preprocess != nil {
		for _, item := range result.Preprocess.Files {
			if item.URI == uri.String() {
				return filename, item.Content
			}
		}
	}
	return filename, content
}

func rangesOverlap(aStart, aEnd, bStart, bEnd int) bool {
	return aStart <= bEnd && bStart <= aEnd
}

func sharedSeverity(value corediagnostic.Severity) diagnostic.Severity {
	switch value {
	case corediagnostic.SeverityError:
		return diagnostic.SeverityError
	case corediagnostic.SeverityWarning:
		return diagnostic.SeverityWarning
	case corediagnostic.SeverityHint:
		return diagnostic.SeverityHint
	default:
		return diagnostic.SeverityInfo
	}
}
