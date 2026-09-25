package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
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

	// Stream 返回 *schema.StreamReader[*schema.Message]
	// 每次 Recv() 拿到一个“消息碎片”（chunk），Content 只是增量文本
	sr, err := cm.Stream(ctx, []*schema.Message{
		schema.SystemMessage("你是一名技术作家。"),
		schema.UserMessage("用 200 字介绍 Go 的channel。"),
	})

	if err != nil {
		log.Fatal("Stream 失败：", err)
	}

	defer sr.Close()

	// 保存所有碎片，最后拼成完整消息
	var chunks []*schema.Message

	thinking := false

	for {
		chunk, err := sr.Recv()
		// 正常结束
		if errors.Is(err, io.EOF) {
			break
		}
		// 网络错误、ctx 超时等
		if err != nil {
			log.Fatal("\n读取流失败：", err)
		}

		chunks = append(chunks, chunk)

		// 思考模式下，推理内容和正文内容是分开流出的
		if chunk.ReasoningContent != "" {
			if !thinking {
				fmt.Print("[思考]")
				thinking = true
			}
			fmt.Print(chunk.ReasoningContent)
		}

		if chunk.Content != "" {
			if thinking {
				fmt.Print("\n[回答]")
				thinking = false
			}
			fmt.Print(chunk.Content)
		}
	}

	fmt.Println()

	// schema.ConcatMessages 把碎片拼成一条完整消息：
	// Content 拼接、ToolCalls 按 Index 合并、Usage 取最后的值……
	full, err := schema.ConcatMessages(chunks)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n共收到 %d 个 chunk，完整回答 %d 字节\n",
		len(chunks), len(full.Content))

	if full.ResponseMeta != nil && full.ResponseMeta.Usage != nil {
		fmt.Println("总 Token：", full.ResponseMeta.Usage.TotalTokens)
	}
}
