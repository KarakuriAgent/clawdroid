package tools

import "fmt"

func init() {
	registerCategoryValidator(validateClipboardParams, "clipboard_copy", "clipboard_read")
}

func validateClipboardParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "clipboard_copy":
		text := toString(args["text"])
		if text == "" {
			return nil, fmt.Errorf("clipboard_copy requires text")
		}
		params["text"] = text

	case "clipboard_read":
		// No params needed
	}

	return params, nil
}
