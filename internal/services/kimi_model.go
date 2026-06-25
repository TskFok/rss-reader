package services

import "strings"

// thinkingParam Kimi thinking 模式参数，见 https://platform.kimi.com/docs/api/models-overview
type thinkingParam struct {
	Type string `json:"type"`
}

func normalizeModelName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func isKimiK26Model(name string) bool {
	return normalizeModelName(name) == "kimi-k2.6"
}

func isKimiK27CodeModel(name string) bool {
	switch normalizeModelName(name) {
	case "kimi-k2.7-code", "kimi-k2.7-code-highspeed":
		return true
	default:
		return false
	}
}

// isKimiFixedSamplingModel 判断是否为采样参数由服务端固定的 Kimi K2.6/K2.7 Code 系列模型。
func isKimiFixedSamplingModel(name string) bool {
	return isKimiK26Model(name) || isKimiK27CodeModel(name)
}

// buildChatCompletionsRequest 按模型特性构造 OpenAI 兼容请求体。
// Kimi K2.6/K2.7 Code 的 temperature、top_p、n、presence_penalty、frequency_penalty
// 均由服务端固定，传入非默认值会报错，因此一律省略。
func buildChatCompletionsRequest(modelName string, maxTokens int, stream bool, messages []chatMessage) chatCompletionsRequest {
	req := chatCompletionsRequest{
		Model:     modelName,
		Messages:  messages,
		MaxTokens: maxTokens,
		Stream:    stream,
	}
	if isKimiK26Model(modelName) {
		// K2.6 支持关闭 thinking；RSS 翻译/总结等任务无需深度推理，关闭可降低耗时与 token 消耗。
		req.Thinking = &thinkingParam{Type: "disabled"}
	}
	// K2.7 Code 始终开启 thinking 且不可禁用，无需（也不应）传 thinking 参数。
	return req
}
