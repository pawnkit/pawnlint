package project_test

import (
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestParseCacheReusesUnchangedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	cache := project.NewParseCache()
	source := project.Source{Path: path, Content: []byte("main() {}\n")}
	parseEvents := 0
	options := project.Options{WorkingDir: dir, ParseCache: cache, ObserveTiming: func(event project.TimingEvent) {
		if event.Stage == project.TimingParse {
			parseEvents++
		}
	}}
	if _, err := project.Build([]project.Source{source}, options); err != nil {
		t.Fatal(err)
	}
	if parseEvents != 1 {
		t.Fatalf("first parse events = %d", parseEvents)
	}
	parseEvents = 0
	if _, err := project.Build([]project.Source{source}, options); err != nil {
		t.Fatal(err)
	}
	if parseEvents != 0 {
		t.Fatalf("cached parse events = %d", parseEvents)
	}
	source.Content = append(source.Content, '\n')
	if _, err := project.Build([]project.Source{source}, options); err != nil {
		t.Fatal(err)
	}
	if parseEvents != 1 {
		t.Fatalf("changed parse events = %d", parseEvents)
	}
}

func TestParseCacheSeparatesTriviaProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	cache := project.NewParseCache()
	source := project.Source{Path: path, Content: []byte("// documentation\nmain() {}\n")}
	parseEvents := 0
	features := project.Features(0)
	options := project.Options{WorkingDir: dir, ParseCache: cache, Features: &features, ObserveTiming: func(event project.TimingEvent) {
		if event.Stage == project.TimingParse {
			parseEvents++
		}
	}}
	if _, err := project.Build([]project.Source{source}, options); err != nil {
		t.Fatal(err)
	}
	features = project.NewFeatures(project.FeatureTrivia)
	if _, err := project.Build([]project.Source{source}, options); err != nil {
		t.Fatal(err)
	}
	if parseEvents != 2 {
		t.Fatalf("parse events = %d, want 2", parseEvents)
	}
}
