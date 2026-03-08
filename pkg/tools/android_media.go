package tools

import "fmt"

func init() {
	registerCategoryValidator(validateMediaParams,
		"media_play_pause", "media_next", "media_previous", "play_music_search")
}

func validateMediaParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "media_play_pause", "media_next", "media_previous":
		// No params needed

	case "play_music_search":
		query := toString(args["query"])
		if query == "" {
			return nil, fmt.Errorf("play_music_search requires query")
		}
		params["query"] = query
	}

	return params, nil
}
