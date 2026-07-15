package analyzer

import "context"

type Source struct {
	Path    string
	Content []byte
}

type Request struct {
	Sources          []Source
	ConfigPath       string
	WorkingDirectory string
	Build            string
}

type Position struct {
	Offset int
	Line   int
	Column int
}

type Range struct {
	Start Position
	End   Position
}

type RelatedLocation struct {
	Range   Range
	Message string
}

type Diagnostic struct {
	RuleID   string
	Code     string
	Severity string
	Category string
	Message  string
	Path     string
	Range    Range
	Related  []RelatedLocation
}

type Edit struct {
	Range   Range
	NewText string
}

type Action struct {
	DiagnosticIndex int
	RuleID          string
	Title           string
	Path            string
	Range           Range
	Edits           []Edit
}

type Result struct {
	Diagnostics []Diagnostic
	SafeEdits   []Action
	Suggestions []Action
	Migrations  []RuleMigration
	Cache       CacheStats
}

type CacheStats struct {
	Hits   int
	Misses int
}

type RuleMigration struct {
	Deprecated  string
	Replacement string
}

func Analyze(ctx context.Context, request Request) (Result, error) {
	return analyze(ctx, request)
}
