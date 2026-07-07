package services

import (
	"strings"

	"github.com/tskfok/rss-reader/internal/models"
)

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

// isKimiFixedSamplingModel 判断是否为需附带固定采样参数的 Kimi K2.6/K2.7 Code 系列模型。
func isKimiFixedSamplingModel(name string) bool {
	return isKimiK26Model(name) || isKimiK27CodeModel(name)
}

func ptrFloat64(v float64) *float64 { return &v }

func ptrInt(v int) *int { return &v }

const (
	defaultKimiTopP             = 0.95
	defaultKimiN                = 1
	defaultKimiPresencePenalty  = 0.0
	defaultKimiFrequencyPenalty = 0.0
)

func resolvedKimiTopP(m *models.AIModel) float64 {
	if m != nil && m.TopP != nil {
		return *m.TopP
	}
	return defaultKimiTopP
}

func resolvedKimiN(m *models.AIModel) int {
	if m != nil && m.N != nil {
		return *m.N
	}
	return defaultKimiN
}

func resolvedKimiPresencePenalty(m *models.AIModel) float64 {
	if m != nil && m.PresencePenalty != nil {
		return *m.PresencePenalty
	}
	return defaultKimiPresencePenalty
}

func resolvedKimiFrequencyPenalty(m *models.AIModel) float64 {
	if m != nil && m.FrequencyPenalty != nil {
		return *m.FrequencyPenalty
	}
	return defaultKimiFrequencyPenalty
}

func applyKimiSamplingParams(req *chatCompletionsRequest, m *models.AIModel) {
	// temperature 由服务端固定，不传；其余参数从模型扩展字段读取，未配置时使用默认值。
	req.TopP = ptrFloat64(resolvedKimiTopP(m))
	req.N = ptrInt(resolvedKimiN(m))
	req.PresencePenalty = ptrFloat64(resolvedKimiPresencePenalty(m))
	req.FrequencyPenalty = ptrFloat64(resolvedKimiFrequencyPenalty(m))
}

// buildChatCompletionsRequest 按模型特性构造 OpenAI 兼容请求体。
func buildChatCompletionsRequest(m *models.AIModel, maxTokens int, stream bool, messages []chatMessage) chatCompletionsRequest {
	modelName := ""
	if m != nil {
		modelName = m.Name
	}
	req := chatCompletionsRequest{
		Model:     modelName,
		Messages:  messages,
		MaxTokens: maxTokens,
		Stream:    stream,
	}
	if isKimiFixedSamplingModel(modelName) {
		applyKimiSamplingParams(&req, m)
	}
	if isKimiK26Model(modelName) {
		// K2.6 支持关闭 thinking；RSS 翻译/总结等任务无需深度推理，关闭可降低耗时与 token 消耗。
		req.Thinking = &thinkingParam{Type: "disabled"}
	}
	// K2.7 Code 始终开启 thinking 且不可禁用，无需（也不应）传 thinking 参数。
	return req
}
