package tools

import "fmt"

func init() {
	registerCategoryValidator(validateNavigationParams,
		"navigate", "search_nearby", "show_map", "get_current_location")
}

func validateNavigationParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "navigate":
		dest := toString(args["destination"])
		if dest == "" {
			return nil, fmt.Errorf("navigate requires destination")
		}
		params["destination"] = dest
		if mode := toString(args["mode"]); mode != "" {
			switch mode {
			case "driving", "walking", "bicycling", "transit":
				params["mode"] = mode
			default:
				return nil, fmt.Errorf("navigate: invalid mode %q (must be driving, walking, bicycling, or transit)", mode)
			}
		}

	case "search_nearby":
		query := toString(args["query"])
		if query == "" {
			return nil, fmt.Errorf("search_nearby requires query")
		}
		params["query"] = query

	case "show_map":
		// At least one of query, latitude+longitude must be provided
		query := toString(args["query"])
		lat, latOk := toFloat64(args["latitude"])
		lng, lngOk := toFloat64(args["longitude"])
		if query == "" && (!latOk || !lngOk) {
			return nil, fmt.Errorf("show_map requires query or latitude+longitude")
		}
		if query != "" {
			params["query"] = query
		}
		if latOk && lngOk {
			params["latitude"] = lat
			params["longitude"] = lng
		}

	case "get_current_location":
		// No params needed
	}

	return params, nil
}
