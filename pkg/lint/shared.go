package lint

import (
	"path/filepath"
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
	files := newSharedDiagnosticFiles(result, path, content)
	for _, item := range result.Diagnostics {
		if !sharedDiagnostic(item.Code) ||
			item.Primary.File != result.File && !sharedProjectDiagnostic(item.Code) {
			continue
		}
		file := files.get(item.Primary.File)
		filename, lines := file.filename, file.lines
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

type sharedDiagnosticFile struct {
	filename string
	lines    *source.LineTable
}

type sharedDiagnosticFiles struct {
	result  *analysis.Result
	path    string
	content []byte
	files   map[coresource.FileID]sharedDiagnosticFile
}

func newSharedDiagnosticFiles(result *analysis.Result, path string, content []byte) *sharedDiagnosticFiles {
	return &sharedDiagnosticFiles{
		result: result, path: path, content: content,
		files: make(map[coresource.FileID]sharedDiagnosticFile),
	}
}

func (f *sharedDiagnosticFiles) get(id coresource.FileID) sharedDiagnosticFile {
	if file, ok := f.files[id]; ok {
		return file
	}
	filename, content := sharedFile(f.result, id, f.path, f.content)
	file := sharedDiagnosticFile{filename: filename, lines: source.NewLineTable(content)}
	f.files[id] = file
	return file
}

func duplicateShared(dst []diagnostic.Diagnostic, code, filename string, start, end int) bool {
	equivalent := sharedRuleID(code)
	if equivalent == "" {
		return false
	}
	for _, item := range dst {
		if item.RuleID == equivalent && sameSharedFile(item.Filename, filename) &&
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
	filename = filepath.Clean(filename)
	if result.Preprocess != nil {
		for _, item := range result.Preprocess.Files {
			if item.URI == uri.String() {
				return filename, item.Content
			}
		}
	}
	return filename, content
}

func sameSharedFile(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
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
