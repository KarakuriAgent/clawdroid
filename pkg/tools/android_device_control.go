package tools

import "fmt"

func init() {
	registerCategoryValidator(validateDeviceControlParams,
		"flashlight", "set_volume", "set_ringer_mode", "set_dnd", "set_brightness")
}

func validateDeviceControlParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "flashlight":
		enabled, ok := toBool(args["enabled"])
		if !ok {
			return nil, fmt.Errorf("flashlight requires enabled (boolean)")
		}
		params["enabled"] = enabled

	case "set_volume":
		stream := toString(args["stream"])
		if stream == "" {
			return nil, fmt.Errorf("set_volume requires stream")
		}
		switch stream {
		case "music", "ring", "notification", "alarm", "system":
			// valid
		default:
			return nil, fmt.Errorf("set_volume: invalid stream %q", stream)
		}
		level, ok := toInt(args["level"])
		if !ok {
			return nil, fmt.Errorf("set_volume requires level (integer)")
		}
		if level < 0 {
			return nil, fmt.Errorf("set_volume: level must be non-negative")
		}
		params["stream"] = stream
		params["level"] = level

	case "set_ringer_mode":
		mode := toString(args["mode"])
		switch mode {
		case "normal", "vibrate", "silent":
			params["mode"] = mode
		default:
			return nil, fmt.Errorf("set_ringer_mode: invalid mode %q (must be normal, vibrate, or silent)", mode)
		}

	case "set_dnd":
		enabled, ok := toBool(args["enabled"])
		if !ok {
			return nil, fmt.Errorf("set_dnd requires enabled (boolean)")
		}
		params["enabled"] = enabled

	case "set_brightness":
		level, ok := toInt(args["level"])
		if !ok {
			return nil, fmt.Errorf("set_brightness requires level (integer)")
		}
		if level < 0 || level > 255 {
			return nil, fmt.Errorf("set_brightness: level must be 0-255, got %d", level)
		}
		params["level"] = level
		if auto, ok := toBool(args["auto"]); ok {
			params["auto"] = auto
		}
	}

	return params, nil
}
