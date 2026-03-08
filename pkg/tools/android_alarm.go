package tools

import "fmt"

func init() {
	registerCategoryValidator(validateAlarmParams,
		"set_alarm", "set_timer", "dismiss_alarm", "show_alarms")
}

func validateAlarmParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "set_alarm":
		hour, hOk := toInt(args["hour"])
		minute, mOk := toInt(args["minute"])
		if !hOk || !mOk {
			return nil, fmt.Errorf("set_alarm requires hour and minute")
		}
		if hour < 0 || hour > 23 {
			return nil, fmt.Errorf("set_alarm: hour must be 0-23, got %d", hour)
		}
		if minute < 0 || minute > 59 {
			return nil, fmt.Errorf("set_alarm: minute must be 0-59, got %d", minute)
		}
		params["hour"] = hour
		params["minute"] = minute
		if msg := toString(args["message"]); msg != "" {
			params["message"] = msg
		}
		if days := toString(args["days"]); days != "" {
			params["days"] = days
		}
		if skipUI, ok := toBool(args["skip_ui"]); ok {
			params["skip_ui"] = skipUI
		}

	case "set_timer":
		duration, ok := toInt(args["duration_seconds"])
		if !ok || duration < 1 || duration > 86400 {
			return nil, fmt.Errorf("set_timer: duration_seconds must be 1-86400")
		}
		params["duration_seconds"] = duration
		if msg := toString(args["message"]); msg != "" {
			params["message"] = msg
		}
		if skipUI, ok := toBool(args["skip_ui"]); ok {
			params["skip_ui"] = skipUI
		}

	case "dismiss_alarm":
		// No params needed

	case "show_alarms":
		// No params needed
	}

	return params, nil
}
