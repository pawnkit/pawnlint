package externalrule_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pawnkit/pawnlint/pkg/externalrule"
)

func TestRunExchangesVersionedJSON(t *testing.T) {
	response, err := externalrule.Run(context.Background(), externalrule.Program{
		Command:     os.Args[0],
		Arguments:   []string{"-test.run=TestExternalRuleHelper"},
		Environment: []string{"PAWNLINT_EXTERNAL_HELPER=valid"},
	}, externalrule.Request{Files: []externalrule.File{{Path: "main.pwn", Content: "main() {}\n"}}, Targets: []string{"main.pwn"}})
	if err != nil {
		t.Fatal(err)
	}
	if response.ProtocolVersion != externalrule.ProtocolVersion || len(response.Diagnostics) != 1 || response.Diagnostics[0].RuleID != "example" {
		t.Fatalf("response = %#v", response)
	}
}

func TestRunRejectsUnknownResponseFields(t *testing.T) {
	_, err := externalrule.Run(context.Background(), externalrule.Program{
		Command:     os.Args[0],
		Arguments:   []string{"-test.run=TestExternalRuleHelper"},
		Environment: []string{"PAWNLINT_EXTERNAL_HELPER=unknown"},
	}, externalrule.Request{})
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunHonorsTimeout(t *testing.T) {
	_, err := externalrule.Run(context.Background(), externalrule.Program{
		Command:     os.Args[0],
		Arguments:   []string{"-test.run=TestExternalRuleHelper"},
		Environment: []string{"PAWNLINT_EXTERNAL_HELPER=slow"},
		Timeout:     10 * time.Millisecond,
	}, externalrule.Request{})
	if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("error = %v", err)
	}
}

func TestExternalRuleHelper(t *testing.T) {
	switch os.Getenv("PAWNLINT_EXTERNAL_HELPER") {
	case "valid":
		var request externalrule.Request
		if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil || request.ProtocolVersion != externalrule.ProtocolVersion {
			os.Exit(2)
		}
		_ = json.NewEncoder(os.Stdout).Encode(externalrule.Response{
			ProtocolVersion: externalrule.ProtocolVersion,
			Diagnostics: []externalrule.Diagnostic{{
				RuleID: "example", Severity: "warning", Category: "style", Message: "example", Path: "main.pwn",
			}},
		})
		os.Exit(0)
	case "unknown":
		_, _ = io.WriteString(os.Stdout, `{"protocolVersion":1,"diagnostics":[],"unknown":true}`)
		os.Exit(0)
	case "slow":
		time.Sleep(time.Second)
		os.Exit(0)
	}
}
