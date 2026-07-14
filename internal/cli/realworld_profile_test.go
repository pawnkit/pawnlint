package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealworldProfile(t *testing.T) {
	root := os.Getenv("PAWNLINT_REALWORLD_ROOT")
	if root == "" {
		t.Skip()
	}
	code := Run([]string{
		"--config=" + filepath.Join(root, ".pawnlint-realworld.toml"),
		"--profile=all",
		"--format=json",
		filepath.Join(root, "gamemodes", "ScavengeSurvive.pwn"),
	}, strings.NewReader(""), io.Discard, io.Discard)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}
