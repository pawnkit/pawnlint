package cli

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/pawnkit/pawnlint/internal/cache"
	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type cacheConfig struct {
	Source     config.File
	Target     string
	Defines    []string
	Enabled    map[string]diagnostic.Severity
	RuleConfig map[string]map[string]any
	Overrides  []config.ResolvedOverride
	Targets    []string
}

func configuredCacheDirectory(r *config.Resolved, projectDir string) string {
	if r.Source.Cache == "" {
		return ""
	}
	directory := r.Source.Cache
	if !filepath.IsAbs(directory) {
		directory = filepath.Join(projectDir, directory)
	}
	return filepath.Clean(directory)
}

func cachedDiagnostics(directory, slotContext, keyContext string, r *config.Resolved, target string, defines []string, api any, model *project.Model, targets []string, analyze func() []diagnostic.Diagnostic) []diagnostic.Diagnostic {
	if directory == "" {
		return analyze()
	}
	cacheSources := make([]cache.Source, 0, len(model.Files))
	seen := make(map[string]struct{}, len(model.Files))
	for _, file := range model.Files {
		path := filepath.Clean(file.Path)
		if _, duplicate := seen[path]; duplicate {
			continue
		}
		seen[path] = struct{}{}
		cacheSources = append(cacheSources, cache.Source{Path: path, Content: file.Source})
	}
	targetPaths := append([]string(nil), targets...)
	for index := range targetPaths {
		targetPaths[index] = filepath.Clean(targetPaths[index])
	}
	sort.Strings(targetPaths)
	key, err := cache.Key(cache.KeyInput{
		Context: keyContext,
		Config: cacheConfig{
			Source:     r.Source,
			Target:     target,
			Defines:    defines,
			Enabled:    r.Enabled,
			RuleConfig: r.RuleConfig,
			Overrides:  r.Overrides,
			Targets:    targetPaths,
		},
		API:     api,
		Sources: cacheSources,
	})
	if err != nil {
		return analyze()
	}
	slot := cache.Slot(slotContext + "\x00" + strings.Join(targetPaths, "\x00"))
	if diagnostics, hit := cache.Load(directory, slot, key); hit && cache.Validate(diagnostics, cacheSources) {
		return diagnostics
	}
	diagnostics := analyze()
	_ = cache.Write(directory, slot, key, diagnostics)
	return diagnostics
}
