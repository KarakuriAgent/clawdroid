package tools

import (
	"fmt"
	"strings"
)

func init() {
	registerCategoryValidator(validateWebParams, "open_url", "web_search")
}

// allowedSchemes is the set of URL schemes allowed for open_url.
var allowedSchemes = map[string]bool{
	"http":  true,
	"https": true,
}

func validateWebParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "open_url":
		rawURL := toString(args["url"])
		if rawURL == "" {
			return nil, fmt.Errorf("open_url requires url")
		}
		scheme := strings.SplitN(rawURL, ":", 2)[0]
		if !allowedSchemes[strings.ToLower(scheme)] {
			return nil, fmt.Errorf("open_url: scheme %q not allowed (only http/https)", scheme)
		}
		params["url"] = rawURL

	case "web_search":
		query := toString(args["query"])
		if query == "" {
			return nil, fmt.Errorf("web_search requires query")
		}
		params["query"] = query
	}

	return params, nil
}
