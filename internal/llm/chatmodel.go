package llm

import (
	"context"
	"eino-tutorial/internal/config"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"
)

// Options 是创建模型时的可选参数
type Options struct {
	Model           string   // 覆盖配置里的模型名（例如给某个 Agent 用更强的模型）
	Temperature     *float32 // 采样温度，nil 表示用模型默认值
	JSONMode        bool     // 要求模型输出 JSON 对象（模型能力：DeepSeek/OpenAI 都支持 json_object）
	DisableThinking bool     // 关闭 DeepSeek 思考模式（DeepSeek 默认开启思考）
}

// NewChatModel 根据 provider 创建 ChatModel。
// 返回值统一是 Eino 的 model.ToolCallingChatModel 接口。
func NewChatModel(ctx context.Context, c config.LLM, o Options) (model.ToolCallingChatModel, error) {
	modelName := c.Model
	if o.Model != "" {
		modelName = o.Model
	}

	switch c.Provider {

	// ① 推荐：用 OpenAI 兼容协议接入 DeepSeek（也可接入任何 OpenAI 兼容服务）
	case "deepseek", "openai":
		cfg := &openai.ChatModelConfig{
			APIKey:      c.APIKey,
			BaseURL:     c.BaseURL, // DeepSeek: https://api.deepseek.com
			Model:       modelName,
			Temperature: o.Temperature,
		}
		if o.JSONMode {
			cfg.ResponseFormat = &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			}
		}
		// thinking 是 DeepSeek 的扩展参数，OpenAI 协议里没有，
		// 所以通过 ExtraFields 原样塞进请求体。
		if o.DisableThinking && c.Provider == "deepseek" {
			cfg.ExtraFields = map[string]any{
				"thinking": map[string]any{"type": "disabled"},
			}
		}
		return openai.NewChatModel(ctx, cfg)

	// ② 使用 eino-ext 的 DeepSeek 专用组件（字段是非指针类型，思考模式有专门字段）
	case "deepseek-native":
		cfg := &deepseek.ChatModelConfig{
			APIKey:  c.APIKey,
			BaseURL: c.BaseURL,
			Model:   modelName,
		}
		if o.Temperature != nil {
			cfg.Temperature = *o.Temperature
		}
		if o.JSONMode {
			cfg.ResponseFormatType = deepseek.ResponseFormatTypeJSONObject
		}
		if o.DisableThinking {
			cfg.ThinkingConfig = &deepseek.ThinkingConfig{Type: "disabled"}
		}
		return deepseek.NewChatModel(ctx, cfg)

	// ③ Gemini 原生协议：需要先创建 genai.Client，再交给 Eino 组件
	case "gemini":
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  c.APIKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			return nil, err
		}
		return gemini.NewChatModel(ctx, &gemini.Config{
			Client:      client,
			Model:       modelName,
			Temperature: o.Temperature,
		})
	}
	return nil, fmt.Errorf("unsupported provider: %s", c.Provider)
}

// F32 是一个小工具：把 float32 字面量变成指针
func F32(v float32) *float32 { return &v }
