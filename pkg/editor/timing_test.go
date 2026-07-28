package editor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	_, err = diagnoseContext(context.Background(), path, content, root, project.NewParseCache(), nil, func(event project.TimingEvent) {
		totals[event.Stage] += event.Duration
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range []project.TimingStage{project.TimingParse, project.TimingPreprocess, project.TimingSemantic} {
		t.Logf("%s took %s", stage, totals[stage])
	}
}
