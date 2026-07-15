package analyzer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/baseline"
	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

type analysisContext struct {
	workingDir      string
	includePaths    []string
	defines         []string
	definesComplete bool
	target          string
	api             *api.Metadata
	entry           string
}

func analyze(ctx context.Context, request Request) (Result, error) {
	if ctx == nil {
		return Result{}, fmt.Errorf("analyzer: nil context")
	}
	if len(request.Sources) == 0 {
		return Result{}, fmt.Errorf("analyzer: no sources")
	}
	reg := rules.Default()
	resolved, projectDir, err := analyzerConfig(request, reg)
	if err != nil {
		return Result{}, err
	}
	provided, err := analyzerSources(request.Sources, request.WorkingDirectory, projectDir)
	if err != nil {
		return Result{}, err
	}
	contexts, variants, err := analyzerContexts(request.Build, request.WorkingDirectory, resolved, projectDir)
	if err != nil {
		return Result{}, err
	}
	perContext := make([][]diagnostic.Diagnostic, 0, len(contexts))
	allSources := make(map[string][]byte)
	for _, settings := range contexts {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		projectSources := append([]project.Source(nil), provided...)
		if settings.entry != "" && !containsProjectSource(projectSources, settings.entry) {
			content, err := os.ReadFile(settings.entry)
			if err != nil {
				return Result{}, fmt.Errorf("analyzer: build entry: %w", err)
			}
			projectSources = append(projectSources, project.Source{Path: settings.entry, Content: content})
		}
		model, err := project.Build(projectSources, project.Options{
			WorkingDir:      settings.workingDir,
			IncludePaths:    settings.includePaths,
			Defines:         settings.defines,
			DefinesComplete: settings.definesComplete,
		})
		if err != nil {
			return Result{}, fmt.Errorf("analyzer: build project: %w", err)
		}
		for _, file := range model.Files {
			allSources[file.Path] = file.Source
		}
		engine := lint.NewEngine(reg)
		engine.Defines = settings.defines
		engine.Target = settings.target
		engine.API = settings.api
		engine.Project = model
		var findings []diagnostic.Diagnostic
		for _, source := range provided {
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
			file := model.File(source.Path)
			if file == nil {
				return Result{}, fmt.Errorf("analyzer: source %q is unavailable in the project", source.Path)
			}
			relative := relativeAnalyzerPath(projectDir, source.Path)
			findings = append(findings, engine.LintProjectFile(file, lint.ProjectAnalysis, resolved.EnabledForPath(relative), resolved.AllKnownRuleIDs, resolved.RuleConfigForPath(relative))...)
		}
		perContext = append(perContext, findings)
	}
	merged := mergeAnalyzerDiagnostics(perContext, variants)
	diagnostic.Sort(merged)
	if resolved.Source.Baseline != "" {
		path := resolved.Source.Baseline
		if !filepath.IsAbs(path) {
			path = filepath.Join(projectDir, path)
		}
		file, err := baseline.Load(filepath.Clean(path))
		if err != nil {
			return Result{}, err
		}
		merged = baseline.Apply(file, merged, allSources, projectDir).Remaining
	}
	if limit := resolved.Source.Lint.MaxDiagnostics; limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}
	result := analyzerResult(merged)
	for _, migration := range resolved.RuleMigrations {
		result.Migrations = append(result.Migrations, RuleMigration{Deprecated: migration.Deprecated, Replacement: migration.Replacement})
	}
	return result, nil
}

func analyzerConfig(request Request, reg *lint.Registrar) (*config.Resolved, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("analyzer: working directory: %w", err)
	}
	file := config.Defaults()
	configPath := ""
	projectDir := cwd
	if request.ConfigPath != "" {
		configPath, err = filepath.Abs(request.ConfigPath)
		if err != nil {
			return nil, "", fmt.Errorf("analyzer: config path: %w", err)
		}
		file, err = config.Load(configPath)
		if err != nil {
			return nil, "", err
		}
		projectDir = filepath.Dir(configPath)
	}
	if request.ConfigPath == "" && request.WorkingDirectory != "" {
		projectDir, err = filepath.Abs(request.WorkingDirectory)
		if err != nil {
			return nil, "", fmt.Errorf("analyzer: working directory: %w", err)
		}
	}
	resolved, err := config.Resolve(file, configPath, reg)
	if err != nil {
		return nil, "", err
	}
	return resolved, filepath.Clean(projectDir), nil
}

func analyzerSources(sources []Source, workingDirectory, projectDir string) ([]project.Source, error) {
	base := projectDir
	if workingDirectory != "" {
		var err error
		base, err = filepath.Abs(workingDirectory)
		if err != nil {
			return nil, fmt.Errorf("analyzer: working directory: %w", err)
		}
	}
	result := make([]project.Source, 0, len(sources))
	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if strings.TrimSpace(source.Path) == "" {
			return nil, fmt.Errorf("analyzer: source path is empty")
		}
		path := source.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, path)
		}
		path = filepath.Clean(path)
		if _, duplicate := seen[path]; duplicate {
			return nil, fmt.Errorf("analyzer: duplicate source %q", path)
		}
		seen[path] = struct{}{}
		result = append(result, project.Source{Path: path, Content: append([]byte(nil), source.Content...)})
	}
	return result, nil
}

func analyzerContexts(buildName, requestedWorkingDirectory string, resolved *config.Resolved, projectDir string) ([]analysisContext, bool, error) {
	if len(resolved.Source.Builds) == 0 {
		if buildName != "" {
			return nil, false, fmt.Errorf("analyzer: unknown build %q", buildName)
		}
		variants := resolved.Source.Variants
		if len(variants) == 0 {
			variants = []config.Variant{{Defines: resolved.Source.Defines}}
		}
		includePaths := resolveAnalyzerPaths(projectDir, resolved.Source.IncludePaths)
		workingDir := projectDir
		if requestedWorkingDirectory != "" {
			var err error
			workingDir, err = filepath.Abs(requestedWorkingDirectory)
			if err != nil {
				return nil, false, fmt.Errorf("analyzer: working directory: %w", err)
			}
		}
		contexts := make([]analysisContext, 0, len(variants))
		for _, variant := range variants {
			contexts = append(contexts, analysisContext{
				workingDir:   workingDir,
				includePaths: includePaths,
				defines:      append([]string(nil), variant.Defines...),
				target:       string(resolved.Target),
				api:          resolved.API,
			})
		}
		return contexts, len(variants) > 1, nil
	}
	var selected []config.Build
	for _, build := range resolved.Source.Builds {
		if buildName == "" || build.Name == buildName {
			selected = append(selected, build)
		}
	}
	if len(selected) == 0 {
		return nil, false, fmt.Errorf("analyzer: unknown build %q", buildName)
	}
	contexts := make([]analysisContext, 0, len(selected))
	for _, build := range selected {
		workingDir := resolveAnalyzerPath(projectDir, build.WorkingDirectory)
		includePaths := resolveAnalyzerPaths(projectDir, resolved.Source.IncludePaths)
		includePaths = appendUniqueAnalyzerPaths(includePaths, resolveAnalyzerPaths(workingDir, build.IncludePaths)...)
		defines := appendUniqueAnalyzerStrings(resolved.Source.Defines, build.Defines...)
		target := resolved.Target
		if build.Target != "" {
			target = config.Target(build.Target)
		}
		metadata, err := resolved.APIForTarget(target)
		if err != nil {
			return nil, false, err
		}
		contexts = append(contexts, analysisContext{
			workingDir:      workingDir,
			includePaths:    includePaths,
			defines:         defines,
			definesComplete: true,
			target:          string(target),
			api:             metadata,
			entry:           resolveAnalyzerPath(workingDir, build.Entry),
		})
	}
	return contexts, false, nil
}

func mergeAnalyzerDiagnostics(contexts [][]diagnostic.Diagnostic, variants bool) []diagnostic.Diagnostic {
	if len(contexts) == 1 {
		return contexts[0]
	}
	type key struct {
		rule       string
		path       string
		start, end int
	}
	seen := make(map[key]struct{})
	firstSuppression := make(map[key]diagnostic.Diagnostic)
	suppressionCount := make(map[key]int)
	var merged []diagnostic.Diagnostic
	for _, findings := range contexts {
		for _, finding := range findings {
			item := key{rule: finding.RuleID, path: finding.Filename, start: finding.Range.Start.Offset, end: finding.Range.End.Offset}
			if variants && finding.RuleID == lint.SuppressionID {
				suppressionCount[item]++
				if _, exists := firstSuppression[item]; !exists {
					firstSuppression[item] = finding
				}
				continue
			}
			if _, exists := seen[item]; exists {
				continue
			}
			seen[item] = struct{}{}
			merged = append(merged, finding)
		}
	}
	if variants {
		for item, count := range suppressionCount {
			if count == len(contexts) {
				merged = append(merged, firstSuppression[item])
			}
		}
	}
	return merged
}

func analyzerResult(findings []diagnostic.Diagnostic) Result {
	result := Result{Diagnostics: make([]Diagnostic, 0, len(findings))}
	for index, finding := range findings {
		converted := Diagnostic{
			RuleID:   finding.RuleID,
			Code:     finding.Code,
			Severity: finding.Severity.String(),
			Category: finding.Category.String(),
			Message:  finding.Message,
			Path:     finding.Filename,
			Range:    analyzerRange(finding.Range.Start.Offset, finding.Range.Start.Line, finding.Range.Start.Col, finding.Range.End.Offset, finding.Range.End.Line, finding.Range.End.Col),
		}
		for _, note := range finding.Notes {
			converted.Related = append(converted.Related, RelatedLocation{
				Range:   analyzerRange(note.Range.Start.Offset, note.Range.Start.Line, note.Range.Start.Col, note.Range.End.Offset, note.Range.End.Line, note.Range.End.Col),
				Message: note.Message,
			})
		}
		result.Diagnostics = append(result.Diagnostics, converted)
		if finding.Fix != nil {
			result.SafeEdits = append(result.SafeEdits, analyzerAction(index, finding, finding.Fix.Description, finding.Fix.Edits))
		}
		for _, suggestion := range finding.Suggestions {
			result.Suggestions = append(result.Suggestions, analyzerAction(index, finding, suggestion.Description, suggestion.Edits))
		}
	}
	return result
}

func analyzerAction(index int, finding diagnostic.Diagnostic, title string, edits []diagnostic.Edit) Action {
	action := Action{DiagnosticIndex: index, RuleID: finding.RuleID, Title: title, Path: finding.Filename, Range: analyzerRange(finding.Range.Start.Offset, finding.Range.Start.Line, finding.Range.Start.Col, finding.Range.End.Offset, finding.Range.End.Line, finding.Range.End.Col)}
	for _, edit := range edits {
		action.Edits = append(action.Edits, Edit{
			Range:   analyzerRange(edit.Range.Start.Offset, edit.Range.Start.Line, edit.Range.Start.Col, edit.Range.End.Offset, edit.Range.End.Line, edit.Range.End.Col),
			NewText: edit.NewText,
		})
	}
	return action
}

func analyzerRange(startOffset, startLine, startColumn, endOffset, endLine, endColumn int) Range {
	return Range{
		Start: Position{Offset: startOffset, Line: startLine, Column: startColumn},
		End:   Position{Offset: endOffset, Line: endLine, Column: endColumn},
	}
}

func resolveAnalyzerPath(base, path string) string {
	if path == "" {
		return filepath.Clean(base)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	return filepath.Clean(path)
}

func resolveAnalyzerPaths(base string, paths []string) []string {
	result := make([]string, len(paths))
	for index, path := range paths {
		result[index] = resolveAnalyzerPath(base, path)
	}
	return result
}

func relativeAnalyzerPath(base, path string) string {
	relative, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(path))
	}
	return filepath.ToSlash(filepath.Clean(relative))
}

func containsProjectSource(sources []project.Source, path string) bool {
	for _, source := range sources {
		if filepath.Clean(source.Path) == filepath.Clean(path) {
			return true
		}
	}
	return false
}

func appendUniqueAnalyzerPaths(values []string, additions ...string) []string {
	result := append([]string(nil), values...)
	seen := make(map[string]struct{}, len(result))
	for _, value := range result {
		seen[filepath.Clean(value)] = struct{}{}
	}
	for _, value := range additions {
		clean := filepath.Clean(value)
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, value)
	}
	return result
}

func appendUniqueAnalyzerStrings(values []string, additions ...string) []string {
	result := append([]string(nil), values...)
	seen := make(map[string]struct{}, len(result))
	for _, value := range result {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
