package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	ctx := context.Background()

	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL: os.Getenv("DEEPSEEK_BASE_URL"),
		Model:   os.Getenv("DEEPSEEK_MODEL"),
	})

	if err != nil {
		log.Fatal("创建模型失败：", err)
	}

	msg, err := cm.Generate(ctx, []*schema.Message{
		schema.UserMessage("你好，请用一句话介绍你自己"),
	})

	if err != nil {
		log.Fatal("调用失败：", err)
	}

	fmt.Println(msg.Content)
}
