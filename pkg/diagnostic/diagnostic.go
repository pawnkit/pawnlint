package diagnostic

import (
	"sort"

	"github.com/pawnkit/pawnlint/internal/source"
)

type Severity uint8

const (
	SeverityOff Severity = iota

	SeverityError

	SeverityWarning

	SeverityInfo

	SeverityHint
)

func (s Severity) String() string {
	switch s {
	case SeverityOff:
		return "off"
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	case SeverityHint:
		return "hint"
	default:
		return "off"
	}
}

func ParseSeverity(s string) (Severity, bool) {
	switch s {
	case "off":
		return SeverityOff, true
	case "error":
		return SeverityError, true
	case "warning":
		return SeverityWarning, true
	case "info":
		return SeverityInfo, true
	case "hint":
		return SeverityHint, true
	default:
		return SeverityOff, false
	}
}

type Category uint8

const (
	CategoryCorrectness Category = iota
	CategorySuspicious
	CategoryPerformance
	CategoryMaintainability
	CategoryOpenMP
	CategoryStyle
	CategorySecurity
	CategoryPortability
	CategoryRestriction
)

func (c Category) String() string {
	switch c {
	case CategoryCorrectness:
		return "correctness"
	case CategorySuspicious:
		return "suspicious"
	case CategoryPerformance:
		return "performance"
	case CategoryMaintainability:
		return "maintainability"
	case CategoryOpenMP:
		return "openmp"
	case CategoryStyle:
		return "style"
	case CategorySecurity:
		return "security"
	case CategoryPortability:
		return "portability"
	case CategoryRestriction:
		return "restriction"
	default:
		return "unknown"
	}
}

type RelatedLocation struct {
	Range   source.Range
	Message string
}

type Diagnostic struct {
	RuleID      string
	Code        string
	Severity    Severity
	Category    Category
	Message     string
	Filename    string
	Range       source.Range
	Notes       []RelatedLocation
	Suggestions []Suggestion
	Fix         *Fix
}

type Suggestion struct {
	Description string
	Edits       []Edit
}

type Fix struct {
	Description string
	Edits       []Edit
}

type Edit struct {
	Range   source.Range
	NewText string
}

func (d *Diagnostic) Add(dst *[]Diagnostic) {
	*dst = append(*dst, *d)
}

func Sort(ds []Diagnostic) {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if a.Filename != b.Filename {
			return a.Filename < b.Filename
		}
		if a.Range.Start.Offset != b.Range.Start.Offset {
			return a.Range.Start.Offset < b.Range.Start.Offset
		}
		if a.RuleID != b.RuleID {
			return a.RuleID < b.RuleID
		}
		return a.Message < b.Message
	})
}
