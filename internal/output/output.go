package output

import (
	"io"

	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

type Format string

const (
	FormatText    Format = "text"
	FormatCompact Format = "compact"
	FormatJSON    Format = "json"
	FormatJSONL   Format = "jsonl"
	FormatSARIF   Format = "sarif"
	FormatGitHub  Format = "github"
)

func AllowedFormat(f string) bool {
	switch Format(f) {
	case FormatText, FormatCompact, FormatJSON, FormatJSONL, FormatSARIF, FormatGitHub:
		return true
	default:
		return false
	}
}

func AllFormats() []string {
	return []string{string(FormatText), string(FormatCompact), string(FormatJSON), string(FormatJSONL), string(FormatSARIF), string(FormatGitHub)}
}

type SourceSet map[string][]byte

func diagnosticID(d diagnostic.Diagnostic) string {
	if d.Code == "" {
		return d.RuleID
	}
	return d.RuleID + "/" + d.Code
}

func (s SourceSet) lineTableFor(filename string) *source.LineTable {
	src := s[filename]
	return source.NewLineTable(src)
}

func Write(w io.Writer, format Format, diags []diagnostic.Diagnostic, sources SourceSet, useColor bool) error {
	switch format {
	case FormatText:
		return writeText(w, diags, sources, useColor)
	case FormatCompact:
		return writeCompact(w, diags)
	case FormatJSON:
		return writeJSON(w, diags, false)
	case FormatJSONL:
		return writeJSON(w, diags, true)
	case FormatSARIF:
		return writeSARIF(w, diags)
	case FormatGitHub:
		return writeGitHub(w, diags)
	default:
		return writeText(w, diags, sources, useColor)
	}
}
