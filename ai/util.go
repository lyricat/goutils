package ai

import "strings"

func supportJSONResponse(model string) bool {
	return strings.HasPrefix(model, "gpt-4") || strings.HasPrefix(model, "gpt-3.5") || strings.HasPrefix(model, "deepseek-chat")
}
