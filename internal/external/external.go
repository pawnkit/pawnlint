package external

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/externalrule"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type Input struct {
	WorkingDirectory string
	Build            string
	Target           string
	Defines          []string
	Files            []externalrule.File
	Targets          []string
	paths            map[string]string
}

func ProjectInput(workingDirectory, build, target string, defines []string, model *project.Model, targetPaths []string) Input {
	input := Input{WorkingDirectory: workingDirectory, Build: build, Target: target, Defines: defines, paths: make(map[string]string)}
	if model != nil {
		for _, file := range model.Files {
			relative := relativePath(workingDirectory, file.Path)
			input.Files = append(input.Files, externalrule.File{Path: relative, Content: string(file.Source)})
			input.paths[relative] = file.Path
		}
	}
	for _, path := range targetPaths {
		input.Targets = append(input.Targets, relativePath(workingDirectory, path))
	}
	return input
}

func SourceInput(workingDirectory, target string, defines []string, path string, content []byte) Input {
	relative := relativePath(workingDirectory, path)
	return Input{
		WorkingDirectory: workingDirectory,
		Target:           target,
		Defines:          defines,
		Files:            []externalrule.File{{Path: relative, Content: string(content)}},
		Targets:          []string{relative},
		paths:            map[string]string{relative: path},
	}
}

func relativePath(base, path string) string {
	if base != "" {
		if relative, err := filepath.Rel(base, path); err == nil {
			return filepath.ToSlash(relative)
		}
	}
	return filepath.ToSlash(filepath.Clean(path))
}

var ruleIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

func Run(ctx context.Context, configured []config.ExternalRule, input Input) ([]diagnostic.Diagnostic, error) {
	if len(configured) == 0 {
		return nil, nil
	}
	files := append([]externalrule.File(nil), input.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	targets := append([]string(nil), input.Targets...)
	sort.Strings(targets)
	request := externalrule.Request{
		ProtocolVersion:  externalrule.ProtocolVersion,
		WorkingDirectory: ".",
		Build:            input.Build,
		Target:           input.Target,
		Defines:          append([]string(nil), input.Defines...),
		Targets:          targets,
		Files:            files,
	}
	var diagnostics []diagnostic.Diagnostic
	for _, configuredRule := range configured {
		command := configuredRule.Command
		if !filepath.IsAbs(command) && strings.ContainsAny(command, `/\`) {
			command = filepath.Join(input.WorkingDirectory, command)
		}
		timeout := time.Duration(configuredRule.TimeoutMS) * time.Millisecond
		programRequest := request
		programRequest.Configuration = configuredRule.Configuration
		response, err := externalrule.Run(ctx, externalrule.Program{
			Command:   command,
			Arguments: configuredRule.Arguments,
			Directory: input.WorkingDirectory,
			Timeout:   timeout,
		}, programRequest)
		if err != nil {
			return nil, fmt.Errorf("external rule %q: %w", configuredRule.Name, err)
		}
		converted, err := convert(configuredRule.Name, files, input.paths, response.Diagnostics)
		if err != nil {
			return nil, fmt.Errorf("external rule %q: %w", configuredRule.Name, err)
		}
		diagnostics = append(diagnostics, converted...)
	}
	diagnostic.Sort(diagnostics)
	return diagnostics, nil
}

func convert(namespace string, files []externalrule.File, paths map[string]string, values []externalrule.Diagnostic) ([]diagnostic.Diagnostic, error) {
	tables := make(map[string]*source.LineTable, len(files))
	lengths := make(map[string]int, len(files))
	for _, file := range files {
		if _, duplicate := tables[file.Path]; duplicate {
			return nil, fmt.Errorf("duplicate request path %q", file.Path)
		}
		content := []byte(file.Content)
		tables[file.Path] = source.NewLineTable(content)
		lengths[file.Path] = len(content)
	}
	result := make([]diagnostic.Diagnostic, 0, len(values))
	for index, value := range values {
		if !ruleIDPattern.MatchString(value.RuleID) {
			return nil, fmt.Errorf("diagnostics[%d] has invalid ruleId %q", index, value.RuleID)
		}
		if strings.TrimSpace(value.Message) == "" {
			return nil, fmt.Errorf("diagnostics[%d] has an empty message", index)
		}
		severity, ok := diagnostic.ParseSeverity(value.Severity)
		if !ok || severity == diagnostic.SeverityOff {
			return nil, fmt.Errorf("diagnostics[%d] has invalid severity %q", index, value.Severity)
		}
		category, ok := parseCategory(value.Category)
		if !ok {
			return nil, fmt.Errorf("diagnostics[%d] has invalid category %q", index, value.Category)
		}
		rangeValue, err := protocolRange(tables, lengths, value.Path, value.StartOffset, value.EndOffset)
		if err != nil {
			return nil, fmt.Errorf("diagnostics[%d]: %w", index, err)
		}
		filename := value.Path
		if paths[value.Path] != "" {
			filename = paths[value.Path]
		}
		converted := diagnostic.Diagnostic{
			RuleID:   "external/" + namespace + "/" + value.RuleID,
			Code:     value.Code,
			Severity: severity,
			Category: category,
			Message:  value.Message,
			Filename: filename,
			Range:    rangeValue,
		}
		for relatedIndex, related := range value.Related {
			if related.Path != value.Path {
				return nil, fmt.Errorf("diagnostics[%d].related[%d] uses a different path", index, relatedIndex)
			}
			relatedRange, err := protocolRange(tables, lengths, related.Path, related.StartOffset, related.EndOffset)
			if err != nil {
				return nil, fmt.Errorf("diagnostics[%d].related[%d]: %w", index, relatedIndex, err)
			}
			converted.Notes = append(converted.Notes, diagnostic.RelatedLocation{Range: relatedRange, Message: related.Message})
		}
		if value.Fix != nil {
			if strings.TrimSpace(value.Fix.Description) == "" || len(value.Fix.Edits) == 0 {
				return nil, fmt.Errorf("diagnostics[%d].fix requires a description and edits", index)
			}
			edits, err := protocolEdits(tables[value.Path], lengths[value.Path], value.Fix.Edits)
			if err != nil {
				return nil, fmt.Errorf("diagnostics[%d].fix: %w", index, err)
			}
			converted.Fix = &diagnostic.Fix{Description: value.Fix.Description, Edits: edits}
		}
		for suggestionIndex, suggestion := range value.Suggestions {
			if strings.TrimSpace(suggestion.Description) == "" {
				return nil, fmt.Errorf("diagnostics[%d].suggestions[%d] has an empty description", index, suggestionIndex)
			}
			edits, err := protocolEdits(tables[value.Path], lengths[value.Path], suggestion.Edits)
			if err != nil {
				return nil, fmt.Errorf("diagnostics[%d].suggestions[%d]: %w", index, suggestionIndex, err)
			}
			converted.Suggestions = append(converted.Suggestions, diagnostic.Suggestion{Description: suggestion.Description, Edits: edits})
		}
		result = append(result, converted)
	}
	return result, nil
}

func protocolRange(tables map[string]*source.LineTable, lengths map[string]int, path string, start, end int) (source.Range, error) {
	table := tables[path]
	length, exists := lengths[path]
	if table == nil || !exists {
		return source.Range{}, fmt.Errorf("unknown path %q", path)
	}
	if start < 0 || end < start || end > length {
		return source.Range{}, fmt.Errorf("invalid offset range %d:%d for %q", start, end, path)
	}
	return table.Range(start, end), nil
}

func protocolEdits(table *source.LineTable, length int, values []externalrule.Edit) ([]diagnostic.Edit, error) {
	result := make([]diagnostic.Edit, 0, len(values))
	for index, value := range values {
		if value.StartOffset < 0 || value.EndOffset < value.StartOffset || value.EndOffset > length {
			return nil, fmt.Errorf("edits[%d] has invalid offset range %d:%d", index, value.StartOffset, value.EndOffset)
		}
		result = append(result, diagnostic.Edit{Range: table.Range(value.StartOffset, value.EndOffset), NewText: value.NewText})
	}
	return result, nil
}

func parseCategory(value string) (diagnostic.Category, bool) {
	for _, category := range []diagnostic.Category{
		diagnostic.CategoryCorrectness,
		diagnostic.CategorySuspicious,
		diagnostic.CategoryPerformance,
		diagnostic.CategoryMaintainability,
		diagnostic.CategoryOpenMP,
		diagnostic.CategoryStyle,
		diagnostic.CategorySecurity,
		diagnostic.CategoryPortability,
		diagnostic.CategoryRestriction,
	} {
		if category.String() == value {
			return category, true
		}
	}
	return 0, false
}
