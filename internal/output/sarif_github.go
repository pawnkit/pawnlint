package output

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name  string      `json:"name"`
	Rules []sarifRule `json:"rules,omitempty"`
}

type sarifRule struct {
	ID               string    `json:"id"`
	ShortDescription sarifText `json:"shortDescription"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID     string           `json:"ruleId"`
	Level      string           `json:"level"`
	Message    sarifText        `json:"message"`
	Locations  []sarifLocation  `json:"locations"`
	Fixes      []sarifFix       `json:"fixes,omitempty"`
	Properties *sarifProperties `json:"properties,omitempty"`
}

type sarifProperties struct {
	Code        string            `json:"code,omitempty"`
	Suggestions []sarifSuggestion `json:"suggestions,omitempty"`
}

type sarifSuggestion struct {
	Description string      `json:"description"`
	Edits       []sarifEdit `json:"edits,omitempty"`
}

type sarifEdit struct {
	StartOffset int    `json:"startOffset"`
	EndOffset   int    `json:"endOffset"`
	NewText     string `json:"newText"`
}

type sarifFix struct {
	Description     sarifText             `json:"description"`
	ArtifactChanges []sarifArtifactChange `json:"artifactChanges"`
}

type sarifArtifactChange struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Replacements     []sarifReplacement    `json:"replacements"`
}

type sarifReplacement struct {
	DeletedRegion   sarifRegion `json:"deletedRegion"`
	InsertedContent sarifText   `json:"insertedContent"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine,omitempty"`
	EndColumn   int `json:"endColumn,omitempty"`
}

func writeSARIF(w io.Writer, diags []diagnostic.Diagnostic) error {
	descriptions := make(map[string]string)
	for _, d := range diags {
		if _, ok := descriptions[d.RuleID]; !ok {
			descriptions[d.RuleID] = d.Message
		}
	}
	ids := make([]string, 0, len(descriptions))
	for id := range descriptions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	rules := make([]sarifRule, 0, len(ids))
	for _, id := range ids {
		rules = append(rules, sarifRule{ID: id, ShortDescription: sarifText{Text: descriptions[id]}})
	}
	results := make([]sarifResult, 0, len(diags))
	for _, d := range diags {
		region := sarifRegion{
			StartLine:   positive(d.Range.Start.Line),
			StartColumn: positive(d.Range.Start.Col),
			EndLine:     d.Range.End.Line,
			EndColumn:   d.Range.End.Col,
		}
		result := sarifResult{
			RuleID:  d.RuleID,
			Level:   sarifLevel(d.Severity),
			Message: sarifText{Text: d.Message},
			Locations: []sarifLocation{{PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{URI: filepath.ToSlash(d.Filename)},
				Region:           region,
			}}},
		}
		if d.Fix != nil {
			replacements := make([]sarifReplacement, 0, len(d.Fix.Edits))
			for _, edit := range d.Fix.Edits {
				replacements = append(replacements, sarifReplacement{
					DeletedRegion: sarifRegion{
						StartLine: positive(edit.Range.Start.Line), StartColumn: positive(edit.Range.Start.Col),
						EndLine: edit.Range.End.Line, EndColumn: edit.Range.End.Col,
					},
					InsertedContent: sarifText{Text: edit.NewText},
				})
			}
			result.Fixes = []sarifFix{{
				Description: sarifText{Text: d.Fix.Description},
				ArtifactChanges: []sarifArtifactChange{{
					ArtifactLocation: sarifArtifactLocation{URI: filepath.ToSlash(d.Filename)},
					Replacements:     replacements,
				}},
			}}
		}
		if d.Code != "" || len(d.Suggestions) != 0 {
			properties := &sarifProperties{Code: d.Code}
			for _, suggestion := range d.Suggestions {
				item := sarifSuggestion{Description: suggestion.Description}
				for _, edit := range suggestion.Edits {
					item.Edits = append(item.Edits, sarifEdit{
						StartOffset: edit.Range.Start.Offset,
						EndOffset:   edit.Range.End.Offset,
						NewText:     edit.NewText,
					})
				}
				properties.Suggestions = append(properties.Suggestions, item)
			}
			result.Properties = properties
		}
		results = append(results, result)
	}
	log := sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool:    sarifTool{Driver: sarifDriver{Name: "pawnlint", Rules: rules}},
			Results: results,
		}},
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(log)
}

func sarifLevel(severity diagnostic.Severity) string {
	switch severity {
	case diagnostic.SeverityError:
		return "error"
	case diagnostic.SeverityWarning:
		return "warning"
	case diagnostic.SeverityInfo, diagnostic.SeverityHint:
		return "note"
	default:
		return "none"
	}
}

func positive(value int) int {
	if value < 1 {
		return 1
	}
	return value
}

func writeGitHub(w io.Writer, diags []diagnostic.Diagnostic) error {
	for _, d := range diags {
		cmd := "notice"
		switch d.Severity {
		case diagnostic.SeverityError:
			cmd = "error"
		case diagnostic.SeverityWarning:
			cmd = "warning"
		}
		start := d.Range.Start
		end := d.Range.End
		properties := fmt.Sprintf("file=%s,line=%d,col=%d", escapeProperty(d.Filename), positive(start.Line), positive(start.Col))
		if end.Line > 0 {
			properties += fmt.Sprintf(",endLine=%d", end.Line)
		}
		if end.Col > 0 && (end.Line == 0 || end.Line == start.Line) {
			properties += fmt.Sprintf(",endColumn=%d", end.Col)
		}
		message := fmt.Sprintf("%s[%s] %s", d.Severity, diagnosticID(d), d.Message)
		if _, err := fmt.Fprintf(w, "::%s %s::%s\n", cmd, properties, escapeMessage(message)); err != nil {
			return err
		}
	}
	return nil
}

func escapeProperty(value string) string {
	r := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")
	return r.Replace(value)
}

func escapeMessage(value string) string {
	r := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	return r.Replace(value)
}
