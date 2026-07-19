// Package editor is the entry point for tools that lint a single open
// buffer with pawnlint's normal config-discovery rules (an LSP server, an
// editor plugin), without linking against pawnlint's internal packages.
package editor

import (
	"path/filepath"

	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

// Diagnose lints content as path, discovering and resolving pawnlint
// configuration starting from workingDir (typically path's directory).
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

	return engine.LintFile(path, content, lint.ProjectAnalysis, resolved.Enabled, resolved.AllKnownRuleIDs, resolved.RuleConfig), nil
}
