package tools

import "fmt"

func init() {
	registerCategoryValidator(validateCalendarParams,
		"create_event", "query_events", "update_event", "delete_event",
		"list_calendars", "add_reminder")
}

func validateCalendarParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "create_event":
		title := toString(args["title"])
		if title == "" {
			return nil, fmt.Errorf("create_event requires title")
		}
		startTime := toString(args["start_time"])
		if startTime == "" {
			return nil, fmt.Errorf("create_event requires start_time")
		}
		params["title"] = title
		params["start_time"] = startTime
		if v := toString(args["end_time"]); v != "" {
			params["end_time"] = v
		}
		if v := toString(args["description"]); v != "" {
			params["description"] = v
		}
		if v := toString(args["location"]); v != "" {
			params["location"] = v
		}
		if v, ok := toBool(args["all_day"]); ok {
			params["all_day"] = v
		}

	case "query_events":
		startTime := toString(args["start_time"])
		endTime := toString(args["end_time"])
		if startTime == "" || endTime == "" {
			return nil, fmt.Errorf("query_events requires start_time and end_time")
		}
		params["start_time"] = startTime
		params["end_time"] = endTime
		if v := toString(args["query"]); v != "" {
			params["query"] = v
		}

	case "update_event":
		eventID := toString(args["event_id"])
		if eventID == "" {
			return nil, fmt.Errorf("update_event requires event_id")
		}
		params["event_id"] = eventID
		for _, key := range []string{"title", "start_time", "end_time", "description", "location"} {
			if v := toString(args[key]); v != "" {
				params[key] = v
			}
		}

	case "delete_event":
		eventID := toString(args["event_id"])
		if eventID == "" {
			return nil, fmt.Errorf("delete_event requires event_id")
		}
		params["event_id"] = eventID

	case "list_calendars":
		// No params needed

	case "add_reminder":
		eventID := toString(args["event_id"])
		if eventID == "" {
			return nil, fmt.Errorf("add_reminder requires event_id")
		}
		minutes, ok := toInt(args["minutes"])
		if !ok || minutes < 0 {
			return nil, fmt.Errorf("add_reminder requires minutes (non-negative integer)")
		}
		params["event_id"] = eventID
		params["minutes"] = minutes
	}

	return params, nil
}
