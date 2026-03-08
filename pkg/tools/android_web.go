package tools

import "fmt"

func init() {
	registerCategoryValidator(validateWebParams, "open_url", "web_search")
}

func validateWebParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "open_url":
		url := toString(args["url"])
		if url == "" {
			return nil, fmt.Errorf("open_url requires url")
		}
		params["url"] = url

	case "web_search":
		query := toString(args["query"])
		if query == "" {
			return nil, fmt.Errorf("web_search requires query")
		}
		params["query"] = query
	}

	return params, nil
}
