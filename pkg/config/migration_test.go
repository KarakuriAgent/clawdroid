package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateConfig_AlreadyCurrent(t *testing.T) {
	cfg := &Config{Version: ConfigVersion}
	if migrateConfig(cfg) {
		t.Error("migrateConfig should return false when already at current version")
	}
}

func TestMigrateConfig_FromZero(t *testing.T) {
	cfg := &Config{}
	if !migrateConfig(cfg) {
		t.Error("migrateConfig should return true when migrating from version 0")
	}
	if cfg.Version != ConfigVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, ConfigVersion)
	}
}

func TestMigrateConfig_FutureVersion(t *testing.T) {
	cfg := &Config{Version: ConfigVersion + 10}
	if migrateConfig(cfg) {
		t.Error("migrateConfig should return false for future versions")
	}
}

func TestLoadConfig_MigratesAndSaves(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	// Write a v0 config (no version field)
	data := `{"llm":{"model":"test-model"},"gateway":{"port":8080}}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Config in memory should be migrated
	if cfg.Version != ConfigVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, ConfigVersion)
	}
	if cfg.LLM.Model != "test-model" {
		t.Errorf("LLM.Model = %q, want %q", cfg.LLM.Model, "test-model")
	}

	// File should have been re-saved with version
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]interface{}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	v, ok := saved["version"]
	if !ok {
		t.Fatal("re-saved config should contain 'version' field")
	}
	if int(v.(float64)) != ConfigVersion {
		t.Errorf("saved version = %v, want %d", v, ConfigVersion)
	}
}

func TestMigrateConfig_FromV1ToV2(t *testing.T) {
	cfg := &Config{Version: 1}
	if !migrateConfig(cfg) {
		t.Error("migrateConfig should return true when migrating from version 1")
	}
	if cfg.Version != ConfigVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, ConfigVersion)
	}
	if !cfg.Agents.Defaults.ShowErrors {
		t.Error("ShowErrors should be true after migration from v1")
	}
	if !cfg.Agents.Defaults.ShowWarnings {
		t.Error("ShowWarnings should be true after migration from v1")
	}
}

func TestMigrateConfig_FromV2ToV3(t *testing.T) {
	cfg := &Config{Version: 2}
	if !migrateConfig(cfg) {
		t.Error("migrateConfig should return true when migrating from version 2")
	}
	if cfg.Version != ConfigVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, ConfigVersion)
	}
	// Categories
	if !cfg.Tools.Android.Alarm.Enabled {
		t.Error("Alarm category should be true after v2->v3 migration")
	}
	if !cfg.Tools.Android.Calendar.Enabled {
		t.Error("Calendar category should be true after v2->v3 migration")
	}
	if cfg.Tools.Android.Contacts.Enabled {
		t.Error("Contacts category should be false (privacy default) after v2->v3 migration")
	}
	if cfg.Tools.Android.Communication.Enabled {
		t.Error("Communication category should be false (privacy default) after v2->v3 migration")
	}
	if !cfg.Tools.Android.Media.Enabled {
		t.Error("Media category should be true after v2->v3 migration")
	}
	// Actions should all be enabled by default
	if !cfg.Tools.Android.Alarm.Actions.SetAlarm {
		t.Error("SetAlarm action should be true after v2->v3 migration")
	}
	if !cfg.Tools.Android.Web.Actions.OpenURL {
		t.Error("OpenURL action should be true after v2->v3 migration")
	}
	if cfg.Tools.Android.DisabledActions != nil {
		t.Error("DisabledActions should be nil after v2->v3 migration")
	}
}

func TestMigrateConfig_FromV2ToV3_WithDisabledActions(t *testing.T) {
	cfg := &Config{
		Version: 2,
		Tools: ToolsConfig{
			Android: AndroidToolsConfig{
				DisabledActions: []string{"open_url", "flashlight", "set_alarm"},
			},
		},
	}
	if !migrateConfig(cfg) {
		t.Error("migrateConfig should return true")
	}
	// Disabled actions should be converted to false
	if cfg.Tools.Android.Web.Actions.OpenURL {
		t.Error("OpenURL should be false (was in DisabledActions)")
	}
	if cfg.Tools.Android.DeviceControl.Actions.Flashlight {
		t.Error("Flashlight should be false (was in DisabledActions)")
	}
	if cfg.Tools.Android.Alarm.Actions.SetAlarm {
		t.Error("SetAlarm should be false (was in DisabledActions)")
	}
	// Other actions should remain true
	if !cfg.Tools.Android.Alarm.Actions.SetTimer {
		t.Error("SetTimer should be true (was not in DisabledActions)")
	}
	if !cfg.Tools.Android.Web.Actions.WebSearch {
		t.Error("WebSearch should be true (was not in DisabledActions)")
	}
	// DisabledActions should be cleared
	if cfg.Tools.Android.DisabledActions != nil {
		t.Error("DisabledActions should be nil after migration")
	}
}

func TestLoadConfig_NoMigrationWhenCurrent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	// Write a config already at current version
	data := `{"version":3,"llm":{"model":"test"}}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	info1, _ := os.Stat(path)
	modTime1 := info1.ModTime()

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Version != ConfigVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, ConfigVersion)
	}

	// File should NOT have been re-saved
	info2, _ := os.Stat(path)
	if info2.ModTime() != modTime1 {
		t.Error("config file should not be re-saved when already at current version")
	}
}

func TestLoadConfig_NonExistent_HasCurrentVersion(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != ConfigVersion {
		t.Errorf("new config Version = %d, want %d", cfg.Version, ConfigVersion)
	}
}
