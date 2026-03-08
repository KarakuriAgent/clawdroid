package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/KarakuriAgent/clawdroid/pkg/config"
	"github.com/google/uuid"
)

const androidToolTimeout = 15 * time.Second

var (
	packageNameRe  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)*$`)
	intentActionRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.]*$`)
)

// SendCallbackWithType is like SendCallback but includes a message type field.
type SendCallbackWithType func(channel, chatID, content, msgType string) error

// toolRequest is the JSON payload sent to the Android device via WebSocket.
type toolRequest struct {
	RequestID string                 `json:"request_id"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params,omitempty"`
}

type AndroidTool struct {
	sendCallback SendCallbackWithType
	channel      string
	chatID       string
	clientType   string
	cfg          config.AndroidToolsConfig
}

func NewAndroidTool(cfg config.AndroidToolsConfig) *AndroidTool {
	return &AndroidTool{cfg: cfg}
}

func (t *AndroidTool) Name() string { return "android" }

// SetClientType restricts available actions based on the connected client.
// "main" (chat mode) hides UI-interaction actions; other values allow all.
func (t *AndroidTool) SetClientType(ct string) {
	t.clientType = ct
}

func (t *AndroidTool) Description() string {
	return buildDescription(enabledActions(t.cfg, t.clientType))
}

func (t *AndroidTool) Parameters() map[string]interface{} {
	return buildParameters(enabledActions(t.cfg, t.clientType))
}

func (t *AndroidTool) SetContext(channel, chatID string) {
	t.channel = channel
	t.chatID = chatID
}

func (t *AndroidTool) SetSendCallback(cb SendCallbackWithType) {
	t.sendCallback = cb
}

func (t *AndroidTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if t.sendCallback == nil {
		return ErrorResult("android tool: send callback not configured")
	}
	if t.channel == "" || t.chatID == "" {
		return ErrorResult("android tool: no active channel context")
	}

	action, _ := args["action"].(string)
	if action == "" {
		return ErrorResult("action is required")
	}

	// Check if action is enabled (category + individual action filter)
	if !t.isActionEnabled(action) {
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}

	// Safety guard: reject UI actions from chat-mode clients
	if t.clientType == "main" && isUIAction(action) {
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}

	params, err := t.validateAndBuildParams(action, args)
	if err != nil {
		return ErrorResult(err.Error())
	}

	return t.sendAndWait(ctx, action, params)
}

func (t *AndroidTool) validateAndBuildParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "search_apps":
		query, _ := args["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("search_apps requires query")
		}
		params["query"] = query

	case "app_info", "launch_app":
		pkg, _ := args["package_name"].(string)
		if pkg == "" {
			return nil, fmt.Errorf("%s requires package_name", action)
		}
		if !packageNameRe.MatchString(pkg) {
			return nil, fmt.Errorf("invalid package_name: %s", pkg)
		}
		params["package_name"] = pkg

	case "screenshot":
		// No params needed

	case "get_ui_tree":
		// Start node selection: resource_id or bounds (mutually exclusive)
		hasResourceID := false
		hasBounds := false
		if rid, ok := args["resource_id"].(string); ok && rid != "" {
			params["resource_id"] = rid
			hasResourceID = true
			if idx, ok := toFloat64(args["index"]); ok {
				idxInt := int(idx)
				if idxInt < 0 {
					return nil, fmt.Errorf("get_ui_tree: index must be non-negative, got %d", idxInt)
				}
				params["index"] = idxInt
			}
		}
		if bx, bxOk := toFloat64(args["bounds_x"]); bxOk {
			if by, byOk := toFloat64(args["bounds_y"]); byOk {
				params["bounds_x"] = bx
				params["bounds_y"] = by
				hasBounds = true
			}
		}
		if hasResourceID && hasBounds {
			return nil, fmt.Errorf("get_ui_tree: cannot specify both resource_id and bounds_x/bounds_y")
		}
		if md, ok := toFloat64(args["max_depth"]); ok {
			params["max_depth"] = int(md)
		}
		if mn, ok := toFloat64(args["max_nodes"]); ok {
			mnInt := int(mn)
			if mnInt < 1 {
				return nil, fmt.Errorf("get_ui_tree: max_nodes must be at least 1, got %d", mnInt)
			}
			params["max_nodes"] = mnInt
		}

	case "tap":
		x, xOk := toFloat64(args["x"])
		y, yOk := toFloat64(args["y"])
		if !xOk || !yOk {
			return nil, fmt.Errorf("tap requires x and y coordinates")
		}
		params["x"] = x
		params["y"] = y

	case "swipe":
		x, xOk := toFloat64(args["x"])
		y, yOk := toFloat64(args["y"])
		x2, x2Ok := toFloat64(args["x2"])
		y2, y2Ok := toFloat64(args["y2"])
		if !xOk || !yOk || !x2Ok || !y2Ok {
			return nil, fmt.Errorf("swipe requires x, y, x2, y2 coordinates")
		}
		params["x"] = x
		params["y"] = y
		params["x2"] = x2
		params["y2"] = y2
		if dur, ok := toFloat64(args["duration_ms"]); ok {
			params["duration_ms"] = int(dur)
		}

	case "text":
		text, _ := args["text"].(string)
		if text == "" {
			return nil, fmt.Errorf("text action requires text parameter")
		}
		params["text"] = text

	case "keyevent":
		key, _ := args["key"].(string)
		if key == "" {
			return nil, fmt.Errorf("keyevent requires key parameter")
		}
		switch key {
		case "back", "home", "recents":
			// valid
		default:
			return nil, fmt.Errorf("invalid key: %s (must be back, home, or recents)", key)
		}
		params["key"] = key

	case "broadcast":
		intentAction, _ := args["intent_action"].(string)
		if intentAction == "" {
			return nil, fmt.Errorf("broadcast requires intent_action")
		}
		if !intentActionRe.MatchString(intentAction) {
			return nil, fmt.Errorf("invalid intent_action: %s", intentAction)
		}
		params["intent_action"] = intentAction
		if extras, ok := args["intent_extras"].(map[string]interface{}); ok {
			params["intent_extras"] = extras
		}

	case "intent":
		intentAction, _ := args["intent_action"].(string)
		if intentAction == "" {
			return nil, fmt.Errorf("intent requires intent_action")
		}
		if !intentActionRe.MatchString(intentAction) {
			return nil, fmt.Errorf("invalid intent_action: %s", intentAction)
		}
		params["intent_action"] = intentAction
		if data, ok := args["intent_data"].(string); ok && data != "" {
			params["intent_data"] = data
		}
		if pkg, ok := args["intent_package"].(string); ok && pkg != "" {
			if !packageNameRe.MatchString(pkg) {
				return nil, fmt.Errorf("invalid intent_package: %s", pkg)
			}
			params["intent_package"] = pkg
		}
		if mimeType, ok := args["intent_type"].(string); ok && mimeType != "" {
			params["intent_type"] = mimeType
		}
		if extras, ok := args["intent_extras"].(map[string]interface{}); ok {
			params["intent_extras"] = extras
		}

	default:
		if fn, ok := categoryValidators[action]; ok {
			return fn(action, args)
		}
		return nil, fmt.Errorf("unknown action: %s", action)
	}

	return params, nil
}

// categoryValidators maps action names to their category-specific validation functions.
// Populated by init() in each android_*.go category file.
var categoryValidators = map[string]func(string, map[string]interface{}) (map[string]interface{}, error){}

// registerCategoryValidator registers a validation function for the given action names.
func registerCategoryValidator(fn func(string, map[string]interface{}) (map[string]interface{}, error), actions ...string) {
	for _, a := range actions {
		categoryValidators[a] = fn
	}
}

// isActionEnabled checks whether an action is allowed by the current config.
func (t *AndroidTool) isActionEnabled(action string) bool {
	cat := actionCategory(action)
	if cat != "" && !isCategoryEnabled(t.cfg, cat) {
		return false
	}
	return !isActionDisabledByConfig(t.cfg, action)
}

// isUIAction returns true if the action is a UI-interaction action.
func isUIAction(action string) bool {
	return uiActionMap[action]
}

// toInt extracts an int from an interface{} (handles float64 and int from JSON).
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}

// toString extracts an optional string from args, returning "" if absent.
func toString(v interface{}) string {
	s, _ := v.(string)
	return s
}

// toBool extracts a bool from an interface{}.
func toBool(v interface{}) (bool, bool) {
	b, ok := v.(bool)
	return b, ok
}

func (t *AndroidTool) sendAndWait(ctx context.Context, action string, params map[string]interface{}) *ToolResult {
	requestID := uuid.New().String()

	req := toolRequest{
		RequestID: requestID,
		Action:    action,
		Params:    params,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to marshal tool request: %v", err))
	}

	// Register waiter before sending to avoid race
	respCh := DeviceResponseWaiter.Register(requestID)

	if err := t.sendCallback(t.channel, t.chatID, string(reqJSON), "tool_request"); err != nil {
		DeviceResponseWaiter.Cleanup(requestID)
		return ErrorResult(fmt.Sprintf("failed to send tool request: %v", err))
	}

	// Wait for response with timeout
	select {
	case content := <-respCh:
		// Check if the response indicates accessibility_required
		if strings.HasPrefix(content, "accessibility_required") {
			return &ToolResult{
				ForUser: "この機能にはユーザー補助の設定が必要です",
				ForLLM:  "accessibility_required: The accessibility service is not enabled. The settings dialog has been shown to the user. Do not retry automatically - wait for the user to enable the service and try again.",
			}
		}
		// Screenshot returns base64 JPEG data — wrap as multimodal result
		if action == "screenshot" {
			return &ToolResult{
				ForLLM: "Screenshot captured.",
				Media:  []string{"data:image/jpeg;base64," + content},
				Silent: true,
			}
		}
		return SilentResult(content)
	case <-time.After(androidToolTimeout):
		DeviceResponseWaiter.Cleanup(requestID)
		return ErrorResult("android tool request timed out (15s)")
	case <-ctx.Done():
		DeviceResponseWaiter.Cleanup(requestID)
		return ErrorResult("android tool request cancelled")
	}
}

// toFloat64 extracts a float64 from an interface{} (handles both float64 and int from JSON).
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
