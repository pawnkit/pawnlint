package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestTaintedDataToSinkIsPreview(t *testing.T) {
	metadata, known := rules.Default().Lookup("tainted-data-to-sink")
	if !known || metadata.Stability != lint.StabilityPreview {
		t.Fatal("tainted data to sink is not preview")
	}
	for _, profile := range lint.AllProfiles() {
		if _, enabled := rules.Default().EnabledForProfile(lint.Profile(profile))["tainted-data-to-sink"]; enabled {
			t.Fatalf("tainted data to sink is enabled by %s", profile)
		}
	}
}
