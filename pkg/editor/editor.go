// Package editor lints open buffers for editor integrations.
package editor

import (
	"path/filepath"

	analysis "github.com/pawnkit/pawn-analysis"
	"github.com/pawnkit/pawnlint/internal/config"
	projectcontext "github.com/pawnkit/pawnlint/internal/project"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

// Diagnose lints content as path using configuration found from workingDir.
func Diagnose(path string, content []byte, workingDir string) ([]diagnostic.Diagnostic, error) {
	return DiagnoseWithCache(path, content, workingDir, nil, nil)
}

// DiagnoseWithCache lints content as path using configuration found from
// workingDir, reusing cache across calls so unchanged includes are not
// re-parsed on every call, and shared (if non-nil) instead of analyzing the
// file again for pawn-analysis:sema/* diagnostics. Nil arguments behave
// exactly like [Diagnose].
func DiagnoseWithCache(path string, content []byte, workingDir string, cache *project.ParseCache, shared *analysis.Result) ([]diagnostic.Diagnostic, error) {
	reg := rules.Default()

	configPath, file, err := config.Discover(workingDir)
	if err != nil {
		return nil, err
	}

	resolved, err := config.Resolve(file, configPath, reg)
	if err != nil {
		return nil, err
	}

	base := workingDir
	if configPath != "" {
		base = filepath.Dir(configPath)
	}

	includePaths := make([]string, len(resolved.Source.IncludePaths))
	for i, p := range resolved.Source.IncludePaths {
		if !filepath.IsAbs(p) {
			p = filepath.Join(base, p)
		}
		includePaths[i] = p
	}
	canonical, err := projectcontext.Canonical(workingDir, includePaths)
	if err != nil {
		return nil, err
	}
	if canonical != nil {
		includePaths = projectcontext.IncludeRoots(canonical)
		workingDir = filepath.FromSlash(canonical.Root())
	}

	features := resolved.ProjectFeatures(reg)
	model, err := project.Build(
		[]project.Source{{Path: path, Content: content}},
		project.Options{
			WorkingDir: workingDir, IncludePaths: includePaths, Defines: resolved.Source.Defines,
			ParseCache: cache, Features: &features,
		},
	)
	if err != nil {
		return nil, err
	}

	engine := lint.NewEngine(reg)
	engine.Defines = resolved.Source.Defines
	engine.Target = string(resolved.Target)
	engine.Project = model
	engine.API = resolved.API
	engine.SharedAnalysis = shared

	diagnostics := engine.LintFile(path, content, lint.ProjectAnalysis, resolved.Enabled, resolved.AllKnownRuleIDs, resolved.RuleConfig)
	diagnostic.Sort(diagnostics)
	return diagnostics, nil
}
