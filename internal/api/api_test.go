package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedNativeMetadata(t *testing.T) {
	if len(openMPNatives) < 500 {
		t.Fatalf("open.mp natives = %d", len(openMPNatives))
	}
	if len(sampNatives) < 300 {
		t.Fatalf("SA-MP natives = %d", len(sampNatives))
	}
	kick := openMPNatives["Kick"]
	if len(kick.Parameters) != 1 || kick.Parameters[0].Default || kick.Parameters[0].Variadic {
		t.Fatalf("Kick metadata = %#v", kick)
	}
	command := openMPNatives["SendRconCommand"]
	if len(command.Parameters) != 2 || !command.Parameters[1].Variadic || command.FormatParameter != 1 {
		t.Fatalf("SendRconCommand metadata = %#v", command)
	}
	if sampNatives["CallLocalFunction"].FormatParameter != 0 || sampNatives["format"].FormatParameter != 3 {
		t.Fatal("format parameter classification is incorrect")
	}
	deprecated := openMPNatives["SendRconCommandf"]
	if deprecated.Deprecated == "" {
		t.Fatal("SendRconCommandf deprecation was not generated")
	}
	playerName := openMPNatives["GetPlayerName"]
	if len(playerName.Buffers) != 1 || playerName.Buffers[0].Parameter != 2 || playerName.Buffers[0].SizeParameter != 3 {
		t.Fatalf("GetPlayerName buffers = %#v", playerName.Buffers)
	}
	if !openMPNatives["SetPlayerAdmin"].OpenMPOnly || openMPNatives["printf"].OpenMPOnly {
		t.Fatal("open.mp-only native classification is incorrect")
	}
	if len(openMPConstants) < 500 || len(sampConstants) < 200 {
		t.Fatalf("constant metadata counts = open.mp %d, SA-MP %d", len(openMPConstants), len(sampConstants))
	}
	if !openMPConstants["CAM_MODE_FIXED"].OpenMPOnly {
		t.Fatal("open.mp-only constant classification is incorrect")
	}
	if openMPNatives["fopen"].Release != "fclose" || sampNatives["db_query"].Release != "db_free_result" {
		t.Fatal("resource release metadata is incorrect")
	}
	if _, ok := Natives("samp")["fopen"]; !ok {
		t.Fatal("shared compiler natives are missing from the SA-MP API")
	}
	unsupported := UnsupportedFunctions("openmp")
	if len(unsupported) != 3 || unsupported["EnableTirePopping"].Suggested == "" || unsupported["SetDeathDropAmount"].Suggested == "" {
		t.Fatalf("unsupported open.mp functions = %#v", unsupported)
	}
	if deprecated := DeprecatedFunctions("openmp"); len(deprecated) != 3 || deprecated["GetPlayerPoolSize"].Suggested == "" {
		t.Fatalf("deprecated open.mp functions = %#v", deprecated)
	}
}

func TestLegacyIncludes(t *testing.T) {
	legacy := LegacyIncludes()
	if len(legacy) != 7 || legacy["a_samp"] != "open.mp" {
		t.Fatalf("legacy includes = %#v", legacy)
	}
}

func TestLoadAndMergeUserMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.json")
	source := `{
  "callbacks": {"OnPluginEvent": {"returnTag": "bool", "parameters": [{"name": "value", "taintSource": "player-input"}]}},
  "natives": {
    "Plugin_Init": {},
    "Plugin_Open": {"returnTag": "PluginHandle", "release": "Plugin_Close", "mustUse": true, "requiresBefore": ["Plugin_Init"]},
    "Plugin_Close": {"parameters": [{"name": "handle", "tag": "PluginHandle", "minimum": 1, "maximum": 8}]},
    "Plugin_Read": {"parameters": [{"name": "result", "arrayRank": 1, "output": true, "taintSource": "network-input"}]},
    "Plugin_Query": {"parameters": [{"name": "query", "arrayRank": 1, "const": true, "taintSink": "sql"}]},
    "Plugin_Clamp": {"pure": true, "parameters": [{"name": "value"}]}
  },
  "functions": {
    "OpenLog": {"returnTag": "File", "release": "CloseLog"},
    "CloseLog": {"parameters": [{"name": "file", "tag": "File", "ownership": "transferred"}]},
    "InspectLog": {"parameters": [{"name": "file", "tag": "File", "ownership": "borrowed"}]},
    "Normalize": {"pure": true, "parameters": [{"name": "value"}]}
  },
  "constants": {"PLUGIN_LIMIT": {}}
}`
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	custom, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := Merge("openmp", custom)
	if err != nil {
		t.Fatal(err)
	}
	closeParameter := metadata.Natives["Plugin_Close"].Parameters[0]
	if metadata.Natives["Plugin_Open"].Release != "Plugin_Close" || !metadata.Natives["Plugin_Open"].MustUse || len(metadata.Natives["Plugin_Open"].RequiresBefore) != 1 || closeParameter.Minimum == nil || *closeParameter.Minimum != 1 || closeParameter.Maximum == nil || *closeParameter.Maximum != 8 || !metadata.Natives["Plugin_Clamp"].Pure || metadata.Natives["Plugin_Read"].Parameters[0].TaintSource != "network-input" || metadata.Natives["Plugin_Query"].Parameters[0].TaintSink != "sql" || metadata.Functions["OpenLog"].Release != "CloseLog" || metadata.Functions["CloseLog"].Parameters[0].Ownership != "transferred" || metadata.Functions["InspectLog"].Parameters[0].Ownership != "borrowed" || !metadata.Functions["Normalize"].Pure || metadata.Callbacks["OnPluginEvent"].Name != "OnPluginEvent" || metadata.Callbacks["OnPluginEvent"].Parameters[0].TaintSource != "player-input" || metadata.Constants["PLUGIN_LIMIT"].Name != "PLUGIN_LIMIT" {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestLoadUserMetadataRejectsInvalidTaintContracts(t *testing.T) {
	for _, source := range []string{
		`{"callbacks":{"Event":{"parameters":[{"taintSource":"Player Input"}]}}}`,
		`{"callbacks":{"Event":{"parameters":[{"taintSource":"player--input"}]}}}`,
		`{"callbacks":{"Event":{"parameters":[{"taintSink":"sql"}]}}}`,
		`{"callbacks":{"Event":{"parameters":[{"output":true,"taintSource":"player-input"}]}}}`,
		`{"natives":{"Read":{"parameters":[{"taintSource":"network-input"}]}}}`,
		`{"natives":{"Read":{"parameters":[{"output":true,"taintSource":"network-input","taintSink":"sql"}]}}}`,
		`{"natives":{"Write":{"parameters":[{"output":true,"taintSink":"sql"}]}}}`,
		`{"functions":{"Write":{"parameters":[{"variadic":true,"taintSink":"format"}]}}}`,
	} {
		path := filepath.Join(t.TempDir(), "api.json")
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("invalid taint contract accepted: %s", source)
		}
	}
}

func TestMergeRejectsInvalidTaintContracts(t *testing.T) {
	metadata := &Metadata{Natives: map[string]Native{"Read": {Parameters: []Parameter{{TaintSource: "network-input"}}}}}
	if _, err := Merge("openmp", metadata); err == nil {
		t.Fatal("invalid merged taint contract was accepted")
	}
}

func TestBuiltinPureNatives(t *testing.T) {
	natives := Natives("openmp")
	if !natives["floatabs"].Pure || !natives["strlen"].Pure || natives["random"].Pure {
		t.Fatal("builtin native purity is incorrect")
	}
}

func TestLoadUserMetadataRejectsInvalidPurity(t *testing.T) {
	for _, source := range []string{
		`{"natives":{"Mutate":{"pure":true,"parameters":[{"output":true}]}}}`,
		`{"natives":{"Acquire":{"pure":true,"release":"Close"},"Close":{}}}`,
		`{"functions":{"Mutate":{"pure":true,"parameters":[{"reference":true}]}}}`,
	} {
		path := filepath.Join(t.TempDir(), "api.json")
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("invalid purity accepted: %s", source)
		}
	}
}

func TestLoadUserMetadataRejectsInvalidOwnership(t *testing.T) {
	for _, source := range []string{
		`{"functions":{"Use":{"parameters":[{"ownership":"shared"}]}}}`,
		`{"functions":{"Use":{"parameters":[{"arrayRank":1,"ownership":"borrowed"}]}}}`,
		`{"functions":{"Use":{"parameters":[{"reference":true,"ownership":"borrowed"}]}}}`,
		`{"functions":{"Use":{"parameters":[{"output":true,"ownership":"transferred"}]}}}`,
		`{"functions":{"Use":{"parameters":[{"variadic":true,"ownership":"transferred"}]}}}`,
	} {
		path := filepath.Join(t.TempDir(), "api.json")
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("invalid ownership accepted: %s", source)
		}
	}
}

func TestLoadUserMetadataRejectsInvalidValueBounds(t *testing.T) {
	for _, source := range []string{
		`{"natives":{"Plugin":{"parameters":[{"minimum":2,"maximum":1}]}}}`,
		`{"natives":{"Plugin":{"parameters":[{"arrayRank":1,"minimum":0}]}}}`,
		`{"natives":{"Plugin":{"parameters":[{"output":true,"maximum":1}]}}}`,
		`{"natives":{"Plugin":{"parameters":[{"variadic":true,"minimum":0}]}}}`,
		`{"natives":{"Plugin":{"parameters":[{"minimum":-2147483649}]}}}`,
		`{"natives":{"Plugin":{"parameters":[{"maximum":2147483648}]}}}`,
	} {
		path := filepath.Join(t.TempDir(), "api.json")
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("invalid value bounds accepted: %s", source)
		}
	}
}

func TestLoadUserMetadataRejectsInvalidCallPrerequisites(t *testing.T) {
	for _, source := range []string{
		`{"natives":{"Plugin":{"requiresBefore":[""]}}}`,
		`{"natives":{"Plugin":{"requiresBefore":["Plugin"]}}}`,
		`{"natives":{"Plugin":{"requiresBefore":["Init","Init"]}}}`,
	} {
		path := filepath.Join(t.TempDir(), "api.json")
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("invalid call prerequisite accepted: %s", source)
		}
	}
}

func TestMergeRejectsInvalidCallPrerequisiteGraph(t *testing.T) {
	for _, metadata := range []*Metadata{
		{Natives: map[string]Native{"Use": {RequiresBefore: []string{"Missing"}}}},
		{Natives: map[string]Native{"First": {RequiresBefore: []string{"Second"}}, "Second": {RequiresBefore: []string{"First"}}}},
	} {
		if _, err := Merge("openmp", metadata); err == nil {
			t.Fatalf("invalid call prerequisite graph accepted: %#v", metadata)
		}
	}
}

func TestMergeRejectsInvalidReleaseRelations(t *testing.T) {
	for _, metadata := range []*Metadata{
		{Functions: map[string]Function{"Open": {Release: "Missing"}}},
		{Functions: map[string]Function{"Open": {Release: "Open"}}},
		{Natives: map[string]Native{"Open": {Release: "Open"}}},
	} {
		if _, err := Merge("openmp", metadata); err == nil {
			t.Fatalf("invalid release relation accepted: %#v", metadata)
		}
	}
}

func TestLoadUserMetadataRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.json")
	if err := os.WriteFile(path, []byte(`{"natives":{"Plugin":{"unknown":true}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("unknown field was accepted")
	}
}
