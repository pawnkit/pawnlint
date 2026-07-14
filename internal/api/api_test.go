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
  "callbacks": {"OnPluginEvent": {"returnTag": "bool", "parameters": [{"name": "value"}]}},
  "natives": {
    "Plugin_Open": {"returnTag": "PluginHandle", "release": "Plugin_Close"},
    "Plugin_Close": {"parameters": [{"name": "handle", "tag": "PluginHandle"}]}
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
	if metadata.Natives["Plugin_Open"].Release != "Plugin_Close" || metadata.Callbacks["OnPluginEvent"].Name != "OnPluginEvent" || metadata.Constants["PLUGIN_LIMIT"].Name != "PLUGIN_LIMIT" {
		t.Fatalf("metadata = %#v", metadata)
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
