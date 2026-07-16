package project_test

import (
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestFeatureDependencies(t *testing.T) {
	features := project.NewFeatures(project.FeatureFunctionEffects)
	for _, feature := range []project.Feature{project.FeatureFunctionEffects, project.FeatureCallGraph, project.FeatureReferences} {
		if !features.Has(feature) {
			t.Fatalf("feature %d is missing", feature)
		}
	}
	if features := project.NewFeatures(project.FeatureRuntimeCalls); !features.Has(project.FeatureRuntimeCalls) || !features.Has(project.FeatureCallGraph) {
		t.Fatal("runtime call dependencies are incomplete")
	}
}

func TestBuildSkipsUnrequestedProjectFeatures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	features := project.Features(0)
	model, err := project.Build([]project.Source{{Path: path, Content: []byte("main() {}\n")}}, project.Options{WorkingDir: dir, Features: &features, ReleaseExpanded: true})
	if err != nil {
		t.Fatal(err)
	}
	if model.CallGraph != nil || len(model.Declarations) != 0 || len(model.Units) != 0 {
		t.Fatalf("model built unrequested features: %#v", model)
	}
}
