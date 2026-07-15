package config_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/pkg/project"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestRecommendedProjectFeatures(t *testing.T) {
	registry := rules.Default()
	resolved, err := config.Resolve(config.File{Profile: "recommended"}, "", registry)
	if err != nil {
		t.Fatal(err)
	}
	features := resolved.ProjectFeatures(registry)
	for _, feature := range []project.Feature{project.FeatureDefinedNames, project.FeatureDuplicates, project.FeatureConflicts, project.FeatureIncludeCycles, project.FeatureIncludeIssues, project.FeatureReferences, project.FeatureCallGraph, project.FeatureFunctionEffects} {
		if !features.Has(feature) {
			t.Fatalf("feature %d is missing", feature)
		}
	}
	if features.Has(project.FeatureUnusedIncludes) {
		t.Fatal("unused include analysis is enabled")
	}
}
