package tools

func init() {
	registerCategoryValidator(validateSettingsParams, "open_settings")
}

var validSettingsSections = map[string]bool{
	"main": true, "wifi": true, "bluetooth": true, "airplane": true,
	"display": true, "sound": true, "battery": true, "apps": true,
	"location": true, "security": true, "accessibility": true,
	"date_time": true, "language": true, "developer": true,
	"about": true, "notification": true, "mobile_data": true,
	"nfc": true, "privacy": true,
}

func validateSettingsParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	section := toString(args["section"])
	if section == "" {
		section = "main"
	}
	if !validSettingsSections[section] {
		section = "main"
	}
	params["section"] = section

	return params, nil
}
