package lint

import (
	"strings"

	analysis "github.com/pawnkit/pawn-analysis"
	corediagnostic "github.com/pawnkit/pawnkit-core/diagnostic"
	coresource "github.com/pawnkit/pawnkit-core/source"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func appendSharedDiagnostics(dst []diagnostic.Diagnostic, path string, content []byte) []diagnostic.Diagnostic {
	result := analysis.Analyze(content, analysis.Options{URI: coresource.FileURI(path)})
	lines := source.NewLineTable(content)
	for _, item := range result.Diagnostics {
		if !strings.HasPrefix(item.Code, "pawn-analysis:sema/") {
			continue
		}
		start, end := int(item.Primary.Start), int(item.Primary.End)
		if duplicateShared(dst, item.Code, start, end) {
			continue
		}
		dst = append(dst, diagnostic.Diagnostic{
			RuleID: item.Code, Code: item.Code, Severity: sharedSeverity(item.Severity),
			Category: diagnostic.CategoryCorrectness, Message: item.Message,
			Filename: path, Range: lines.Range(start, end),
		})
	}
	return dst
}

func duplicateShared(dst []diagnostic.Diagnostic, code string, start, end int) bool {
	equivalent := map[string]string{
		"pawn-analysis:sema/not-callable":   "non-callable-symbol",
		"pawn-analysis:sema/tag-mismatch":   "argument-tag-mismatch",
		"pawn-analysis:sema/unreachable":    "unreachable-code",
		"pawn-analysis:sema/missing-return": "missing-return-value",
	}[code]
	if equivalent == "" {
		return false
	}
	for _, item := range dst {
		if item.RuleID == equivalent && rangesOverlap(start, end, item.Range.Start.Offset, item.Range.End.Offset) {
			return true
		}
	}
	return false
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
