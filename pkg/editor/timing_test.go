package editor

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestRealProjectEditorStageLatency(t *testing.T) {
	root := os.Getenv("PAWN_REAL_PROJECT_DIR")
	if root == "" {
		t.Skip("PAWN_REAL_PROJECT_DIR is not set")
	}
	path := filepath.Join(root, "src", "safw.pwn")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	totals := make(map[project.TimingStage]time.Duration)
	rules := make(map[string]time.Duration)
	_, err = diagnoseContextWithTimings(
		context.Background(), path, content, root, project.NewParseCache(), nil,
		func(event project.TimingEvent) {
			totals[event.Stage] += event.Duration
		},
		func(event lint.TimingEvent) {
			if event.Stage == lint.TimingRule {
				rules[event.RuleID] += event.Duration
			}
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range []project.TimingStage{project.TimingParse, project.TimingPreprocess, project.TimingSemantic} {
		t.Logf("%s took %s", stage, totals[stage])
	}
	type ruleTiming struct {
		id       string
		duration time.Duration
	}
	ordered := make([]ruleTiming, 0, len(rules))
	for id, duration := range rules {
		ordered = append(ordered, ruleTiming{id: id, duration: duration})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].duration > ordered[j].duration })
	for _, item := range ordered[:min(10, len(ordered))] {
		t.Logf("rule %s took %s", item.id, item.duration)
	}
}
