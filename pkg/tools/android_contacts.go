package tools

import "fmt"

func init() {
	registerCategoryValidator(validateContactsParams,
		"search_contacts", "get_contact_detail", "add_contact")
}

func validateContactsParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "search_contacts":
		query := toString(args["query"])
		if query == "" {
			return nil, fmt.Errorf("search_contacts requires query")
		}
		params["query"] = query

	case "get_contact_detail":
		contactID := toString(args["contact_id"])
		if contactID == "" {
			return nil, fmt.Errorf("get_contact_detail requires contact_id")
		}
		params["contact_id"] = contactID

	case "add_contact":
		name := toString(args["name"])
		if name == "" {
			return nil, fmt.Errorf("add_contact requires name")
		}
		params["name"] = name
		if v := toString(args["phone"]); v != "" {
			params["phone"] = v
		}
		if v := toString(args["email"]); v != "" {
			params["email"] = v
		}
	}

	return params, nil
}
