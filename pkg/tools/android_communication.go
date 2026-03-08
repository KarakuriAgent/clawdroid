package tools

import "fmt"

func init() {
	registerCategoryValidator(validateCommunicationParams,
		"dial", "compose_sms", "compose_email")
}

func validateCommunicationParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "dial":
		phone := toString(args["phone_number"])
		if phone == "" {
			return nil, fmt.Errorf("dial requires phone_number")
		}
		params["phone_number"] = phone

	case "compose_sms":
		phone := toString(args["phone_number"])
		if phone == "" {
			return nil, fmt.Errorf("compose_sms requires phone_number")
		}
		params["phone_number"] = phone
		if v := toString(args["message"]); v != "" {
			params["message"] = v
		}

	case "compose_email":
		to := toString(args["to"])
		if to == "" {
			return nil, fmt.Errorf("compose_email requires to")
		}
		params["to"] = to
		if v := toString(args["subject"]); v != "" {
			params["subject"] = v
		}
		if v := toString(args["body"]); v != "" {
			params["body"] = v
		}
	}

	return params, nil
}
