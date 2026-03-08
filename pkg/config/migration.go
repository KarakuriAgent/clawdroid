package config

// ConfigVersion is the current config schema version.
// Increment this when adding new fields that must appear in existing config files.
const ConfigVersion = 3

// migrateConfig runs version-based migrations on cfg.
// Returns true if migrations were applied and config should be re-saved.
func migrateConfig(cfg *Config) bool {
	if cfg.Version >= ConfigVersion {
		return false
	}

	migrations := []func(*Config){
		migrateV0ToV1,
		migrateV1ToV2,
		migrateV2ToV3,
	}

	for i := cfg.Version; i < ConfigVersion && i < len(migrations); i++ {
		migrations[i](cfg)
	}

	cfg.Version = ConfigVersion
	return true
}

func migrateV0ToV1(cfg *Config) {
	// queue_messages: Go zero value (false) is the correct default.
	// Version bump + re-save writes the new field into config.json.
}

func migrateV1ToV2(cfg *Config) {
	cfg.Agents.Defaults.ShowErrors = true
	cfg.Agents.Defaults.ShowWarnings = true
}

func migrateV2ToV3(cfg *Config) {
	// Start from defaults (all actions enabled, privacy categories off).
	def := DefaultAndroidToolsConfig()
	// Preserve the existing Enabled flag.
	def.Enabled = cfg.Tools.Android.Enabled

	// Convert old DisabledActions to individual action bools.
	disabled := make(map[string]bool, len(cfg.Tools.Android.DisabledActions))
	for _, d := range cfg.Tools.Android.DisabledActions {
		disabled[d] = true
	}
	if len(disabled) > 0 {
		disableActions(&def, disabled)
	}
	def.DisabledActions = nil

	cfg.Tools.Android = def
}

// disableActions sets action fields to false for any action name present in disabled.
func disableActions(cfg *AndroidToolsConfig, disabled map[string]bool) {
	// Alarm
	if disabled["set_alarm"] {
		cfg.Alarm.Actions.SetAlarm = false
	}
	if disabled["set_timer"] {
		cfg.Alarm.Actions.SetTimer = false
	}
	if disabled["dismiss_alarm"] {
		cfg.Alarm.Actions.DismissAlarm = false
	}
	if disabled["show_alarms"] {
		cfg.Alarm.Actions.ShowAlarms = false
	}
	// Calendar
	if disabled["create_event"] {
		cfg.Calendar.Actions.CreateEvent = false
	}
	if disabled["query_events"] {
		cfg.Calendar.Actions.QueryEvents = false
	}
	if disabled["update_event"] {
		cfg.Calendar.Actions.UpdateEvent = false
	}
	if disabled["delete_event"] {
		cfg.Calendar.Actions.DeleteEvent = false
	}
	if disabled["list_calendars"] {
		cfg.Calendar.Actions.ListCalendars = false
	}
	if disabled["add_reminder"] {
		cfg.Calendar.Actions.AddReminder = false
	}
	// Contacts
	if disabled["search_contacts"] {
		cfg.Contacts.Actions.SearchContacts = false
	}
	if disabled["get_contact_detail"] {
		cfg.Contacts.Actions.GetContactDetail = false
	}
	if disabled["add_contact"] {
		cfg.Contacts.Actions.AddContact = false
	}
	// Communication
	if disabled["dial"] {
		cfg.Communication.Actions.Dial = false
	}
	if disabled["compose_sms"] {
		cfg.Communication.Actions.ComposeSMS = false
	}
	if disabled["compose_email"] {
		cfg.Communication.Actions.ComposeEmail = false
	}
	// Media
	if disabled["media_play_pause"] {
		cfg.Media.Actions.PlayPause = false
	}
	if disabled["media_next"] {
		cfg.Media.Actions.Next = false
	}
	if disabled["media_previous"] {
		cfg.Media.Actions.Previous = false
	}
	if disabled["play_music_search"] {
		cfg.Media.Actions.PlayMusicSearch = false
	}
	// Navigation
	if disabled["navigate"] {
		cfg.Navigation.Actions.Navigate = false
	}
	if disabled["search_nearby"] {
		cfg.Navigation.Actions.SearchNearby = false
	}
	if disabled["show_map"] {
		cfg.Navigation.Actions.ShowMap = false
	}
	if disabled["get_current_location"] {
		cfg.Navigation.Actions.GetCurrentLocation = false
	}
	// Device Control
	if disabled["flashlight"] {
		cfg.DeviceControl.Actions.Flashlight = false
	}
	if disabled["set_volume"] {
		cfg.DeviceControl.Actions.SetVolume = false
	}
	if disabled["set_ringer_mode"] {
		cfg.DeviceControl.Actions.SetRingerMode = false
	}
	if disabled["set_dnd"] {
		cfg.DeviceControl.Actions.SetDND = false
	}
	if disabled["set_brightness"] {
		cfg.DeviceControl.Actions.SetBrightness = false
	}
	// Settings
	if disabled["open_settings"] {
		cfg.Settings.Actions.OpenSettings = false
	}
	// Web
	if disabled["open_url"] {
		cfg.Web.Actions.OpenURL = false
	}
	if disabled["web_search"] {
		cfg.Web.Actions.WebSearch = false
	}
	// Clipboard
	if disabled["clipboard_copy"] {
		cfg.Clipboard.Actions.ClipboardCopy = false
	}
	if disabled["clipboard_read"] {
		cfg.Clipboard.Actions.ClipboardRead = false
	}
}
