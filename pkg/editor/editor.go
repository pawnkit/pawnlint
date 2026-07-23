// Package editor lints open buffers for editor integrations.
package editor

import (
	"path/filepath"

	"github.com/pawnkit/pawnlint/internal/config"
	projectcontext "github.com/pawnkit/pawnlint/internal/project"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

// Diagnose lints content as path using configuration found from workingDir.
func Diagnose(path string, content []byte, workingDir string) ([]diagnostic.Diagnostic, error) {
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

	model, err := project.Build(
		[]project.Source{{Path: path, Content: content}},
		project.Options{WorkingDir: workingDir, IncludePaths: includePaths, Defines: resolved.Source.Defines},
	)
	if err != nil {
		return nil, err
	}

	engine := lint.NewEngine(reg)
	engine.Defines = resolved.Source.Defines
	engine.Target = string(resolved.Target)
	engine.Project = model
	engine.API = resolved.API

	diagnostics := engine.LintFile(path, content, lint.ProjectAnalysis, resolved.Enabled, resolved.AllKnownRuleIDs, resolved.RuleConfig)
	diagnostic.Sort(diagnostics)
	return diagnostics, nil
}
