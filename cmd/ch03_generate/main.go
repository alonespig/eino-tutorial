package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func main() {

	_ = godotenv.Load()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	defer cancel()

	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL:     os.Getenv("DEEPSEEK_BASE_URL"), // DeepSeek: https://api.deepseek.com
		Model:       os.Getenv("DEEPSEEK_MODEL"),
		Temperature: new(float32(0.3)),
	})

	if err != nil {
		log.Fatal(err)
	}

	messages := []*schema.Message{
		schema.SystemMessage("你是一名资深 Go 工程是，回答简洁、准确，必要时给出代码。"),
		schema.UserMessage("sync.Mutex 和 sync.RWMutex 应该怎么选？"),
	}

	resp, err := cm.Generate(ctx, messages,
		model.WithTemperature(0.2),
		model.WithMaxTokens(800),
	)

	if err != nil {
		log.Fatal("Generate失败：", err)
	}

	fmt.Println("角色：", resp.Role)
	if resp.ReasoningContent != "" {
		fmt.Println("-- 推理过程 --\n", resp.ReasoningContent)
	}
	fmt.Println("-- 回答 --\n" + resp.Content)

	if meta := resp.ResponseMeta; meta != nil {
		fmt.Println("结束原因：", meta.FinishReason)
		if u := meta.Usage; u != nil {
			fmt.Printf("Token：输出 %d, 输出 %d, 合计 %d\n",
				u.PromptTokens,
				u.CompletionTokens,
				u.TotalTokens)
		}
	}
}
