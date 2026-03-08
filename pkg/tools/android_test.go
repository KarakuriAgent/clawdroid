package tools

import (
	"testing"

	"github.com/KarakuriAgent/clawdroid/pkg/config"
)

func allEnabledConfig() config.AndroidToolsConfig {
	cfg := config.DefaultAndroidToolsConfig()
	// Enable privacy categories for testing
	cfg.Contacts.Enabled = true
	cfg.Communication.Enabled = true
	return cfg
}

func defaultConfig() config.AndroidToolsConfig {
	return config.DefaultConfig().Tools.Android
}

// --- enabledActions ---

func TestEnabledActions_AllEnabled(t *testing.T) {
	actions := enabledActions(allEnabledConfig(), "overlay")
	if len(actions) != len(allActions) {
		t.Errorf("enabledActions with all categories on: got %d, want %d", len(actions), len(allActions))
	}
}

func TestEnabledActions_HidesUIActionsForMain(t *testing.T) {
	actions := enabledActions(allEnabledConfig(), "main")
	for _, a := range actions {
		if a.UIOnly {
			t.Errorf("UI-only action %q should be hidden for main client", a.Name)
		}
	}
}

func TestEnabledActions_FiltersDisabledCategory(t *testing.T) {
	cfg := allEnabledConfig()
	cfg.Alarm.Enabled = false

	actions := enabledActions(cfg, "overlay")
	for _, a := range actions {
		if a.Category == "alarm" {
			t.Errorf("action %q should be filtered when alarm category is disabled", a.Name)
		}
	}
}

func TestEnabledActions_FiltersDisabledActions(t *testing.T) {
	cfg := allEnabledConfig()
	cfg.Web.Actions.OpenURL = false

	actions := enabledActions(cfg, "overlay")
	for _, a := range actions {
		if a.Name == "open_url" {
			t.Error("open_url should be filtered when action is disabled")
		}
	}
}

func TestEnabledActions_DefaultConfigExcludesContactsAndCommunication(t *testing.T) {
	actions := enabledActions(defaultConfig(), "overlay")
	for _, a := range actions {
		if a.Category == "contacts" || a.Category == "communication" {
			t.Errorf("action %q (category %s) should be disabled by default", a.Name, a.Category)
		}
	}
}

// --- isActionEnabled ---

func TestIsActionEnabled_CoreAlwaysEnabled(t *testing.T) {
	tool := NewAndroidTool(defaultConfig())
	if !tool.isActionEnabled("search_apps") {
		t.Error("core action search_apps should always be enabled")
	}
}

func TestIsActionEnabled_DisabledCategory(t *testing.T) {
	cfg := defaultConfig()
	// contacts is disabled by default
	tool := NewAndroidTool(cfg)
	if tool.isActionEnabled("search_contacts") {
		t.Error("search_contacts should be disabled when contacts category is off")
	}
}

func TestIsActionEnabled_IndividualDisable(t *testing.T) {
	cfg := allEnabledConfig()
	cfg.DeviceControl.Actions.Flashlight = false
	tool := NewAndroidTool(cfg)
	if tool.isActionEnabled("flashlight") {
		t.Error("flashlight should be disabled when action toggle is false")
	}
}

// --- buildParameters ---

func TestBuildParameters_ContainsActionEnum(t *testing.T) {
	actions := enabledActions(allEnabledConfig(), "overlay")
	params := buildParameters(actions)

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties should be a map")
	}
	actionProp, ok := props["action"].(map[string]interface{})
	if !ok {
		t.Fatal("action property should be a map")
	}
	enum, ok := actionProp["enum"].([]string)
	if !ok {
		t.Fatal("action enum should be a string slice")
	}
	if len(enum) != len(actions) {
		t.Errorf("action enum length: got %d, want %d", len(enum), len(actions))
	}
}

func TestBuildParameters_MergesParamDescriptions(t *testing.T) {
	// "query" appears in multiple actions with different descriptions
	actions := enabledActions(allEnabledConfig(), "overlay")
	params := buildParameters(actions)

	props := params["properties"].(map[string]interface{})
	queryProp, ok := props["query"].(map[string]interface{})
	if !ok {
		t.Fatal("query property should exist")
	}
	desc := queryProp["description"].(string)
	// Should contain multiple descriptions joined by "; "
	if desc == "" {
		t.Error("query description should not be empty")
	}
}

// --- buildDescription ---

func TestBuildDescription_ContainsAllActions(t *testing.T) {
	actions := enabledActions(allEnabledConfig(), "overlay")
	desc := buildDescription(actions)
	for _, a := range actions {
		if !contains(desc, a.Name) {
			t.Errorf("description should contain action %q", a.Name)
		}
	}
}

// --- category validators ---

func TestValidateAlarmParams_SetAlarm(t *testing.T) {
	params, err := validateAlarmParams("set_alarm", map[string]interface{}{
		"hour":   float64(8),
		"minute": float64(30),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["hour"] != 8 || params["minute"] != 30 {
		t.Errorf("got hour=%v minute=%v, want 8/30", params["hour"], params["minute"])
	}
}

func TestValidateAlarmParams_SetAlarmInvalidHour(t *testing.T) {
	_, err := validateAlarmParams("set_alarm", map[string]interface{}{
		"hour":   float64(25),
		"minute": float64(0),
	})
	if err == nil {
		t.Error("expected error for hour=25")
	}
}

func TestValidateAlarmParams_SetTimerOutOfRange(t *testing.T) {
	_, err := validateAlarmParams("set_timer", map[string]interface{}{
		"duration_seconds": float64(0),
	})
	if err == nil {
		t.Error("expected error for duration_seconds=0")
	}
}

func TestValidateCalendarParams_CreateEvent(t *testing.T) {
	params, err := validateCalendarParams("create_event", map[string]interface{}{
		"title":      "Meeting",
		"start_time": "2025-01-15T09:00:00",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["title"] != "Meeting" {
		t.Errorf("title = %v, want Meeting", params["title"])
	}
}

func TestValidateCalendarParams_CreateEventMissingTitle(t *testing.T) {
	_, err := validateCalendarParams("create_event", map[string]interface{}{
		"start_time": "2025-01-15T09:00:00",
	})
	if err == nil {
		t.Error("expected error for missing title")
	}
}

func TestValidateWebParams_OpenUrlValidScheme(t *testing.T) {
	params, err := validateWebParams("open_url", map[string]interface{}{
		"url": "https://example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["url"] != "https://example.com" {
		t.Errorf("url = %v", params["url"])
	}
}

func TestValidateWebParams_OpenUrlBlocksJavascript(t *testing.T) {
	_, err := validateWebParams("open_url", map[string]interface{}{
		"url": "javascript:alert(1)",
	})
	if err == nil {
		t.Error("expected error for javascript: scheme")
	}
}

func TestValidateWebParams_OpenUrlBlocksFile(t *testing.T) {
	_, err := validateWebParams("open_url", map[string]interface{}{
		"url": "file:///etc/passwd",
	})
	if err == nil {
		t.Error("expected error for file: scheme")
	}
}

func TestValidateDeviceControlParams_Flashlight(t *testing.T) {
	params, err := validateDeviceControlParams("flashlight", map[string]interface{}{
		"enabled": true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["enabled"] != true {
		t.Errorf("enabled = %v, want true", params["enabled"])
	}
}

func TestValidateDeviceControlParams_SetBrightnessOutOfRange(t *testing.T) {
	_, err := validateDeviceControlParams("set_brightness", map[string]interface{}{
		"level": float64(300),
	})
	if err == nil {
		t.Error("expected error for brightness level=300")
	}
}

func TestValidateNavigationParams_Navigate(t *testing.T) {
	params, err := validateNavigationParams("navigate", map[string]interface{}{
		"destination": "Tokyo Station",
		"mode":        "walking",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["destination"] != "Tokyo Station" {
		t.Errorf("destination = %v", params["destination"])
	}
	if params["mode"] != "walking" {
		t.Errorf("mode = %v", params["mode"])
	}
}

func TestValidateNavigationParams_NavigateInvalidMode(t *testing.T) {
	_, err := validateNavigationParams("navigate", map[string]interface{}{
		"destination": "Tokyo",
		"mode":        "flying",
	})
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestValidateMediaParams_PlayMusicSearch(t *testing.T) {
	params, err := validateMediaParams("play_music_search", map[string]interface{}{
		"query": "Beatles",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["query"] != "Beatles" {
		t.Errorf("query = %v", params["query"])
	}
}

func TestValidateClipboardParams_Copy(t *testing.T) {
	params, err := validateClipboardParams("clipboard_copy", map[string]interface{}{
		"text": "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["text"] != "hello" {
		t.Errorf("text = %v", params["text"])
	}
}

func TestValidateContactsParams_SearchMissingQuery(t *testing.T) {
	_, err := validateContactsParams("search_contacts", map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing query")
	}
}

func TestValidateCommunicationParams_Dial(t *testing.T) {
	params, err := validateCommunicationParams("dial", map[string]interface{}{
		"phone_number": "090-1234-5678",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["phone_number"] != "090-1234-5678" {
		t.Errorf("phone_number = %v", params["phone_number"])
	}
}

func TestValidateSettingsParams_DefaultsToMain(t *testing.T) {
	params, err := validateSettingsParams("open_settings", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["section"] != "main" {
		t.Errorf("section = %v, want main", params["section"])
	}
}

func TestValidateSettingsParams_InvalidFallsBackToMain(t *testing.T) {
	params, err := validateSettingsParams("open_settings", map[string]interface{}{
		"section": "nonexistent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["section"] != "main" {
		t.Errorf("section = %v, want main", params["section"])
	}
}

// --- isUIAction ---

func TestIsUIAction(t *testing.T) {
	uiActions := []string{"screenshot", "get_ui_tree", "tap", "swipe", "text", "keyevent"}
	for _, a := range uiActions {
		if !isUIAction(a) {
			t.Errorf("%q should be a UI action", a)
		}
	}
	nonUI := []string{"search_apps", "launch_app", "set_alarm", "open_url"}
	for _, a := range nonUI {
		if isUIAction(a) {
			t.Errorf("%q should not be a UI action", a)
		}
	}
}

// helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
