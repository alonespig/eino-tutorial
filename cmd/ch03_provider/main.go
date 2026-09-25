package main

import (
	"context"
	"eino-tutorial/internal/config"
	"eino-tutorial/internal/llm"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func main() {
	cfg := config.MustLoad()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cm, err := llm.NewChatModel(ctx, cfg.LLM, llm.Options{
		DisableThinking: false,
	})

	if err != nil {
		log.Fatal(err)
	}

	ans, err := ask(ctx, cm, "用一句话说明你是哪家公司的模型")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("[%s / %s] %s\n", cfg.LLM.Provider, cfg.LLM.Model, ans)
}

func ask(ctx context.Context, cm model.BaseChatModel, q string) (string, error) {
	msg, err := cm.Generate(ctx, []*schema.Message{
		schema.UserMessage(q),
	})

	if err != nil {
		return "", err
	}
	return msg.Content, nil
}
