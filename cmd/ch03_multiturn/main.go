package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
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

	history := []*schema.Message{
		schema.SystemMessage("你是一个友好的中文助手。"),
	}

	in := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("输入问题，输入exit退出\n-> ")
		if !in.Scan() {
			return
		}

		q := strings.TrimSpace(in.Text())

		if q == "" {
			continue
		}

		if q == "exit" {
			return
		}

		history = append(history, schema.UserMessage(q))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		reply, err := streamOnce(ctx, cm, history)
		cancel()

		if err != nil {
			fmt.Println("\n出错：", err)
			// 失败则撤回本轮用户消息
			history = history[:len(history)-1]
			continue
		}

		// 把完整的 assistant 消息追加进历史（包括 ReasoningContent，后续章节会解释为什么要保留）
		history = append(history, reply)
		fmt.Printf("\n（当前历史 %d 条消息）\n", len(history))
	}
}

// streamOnce 流式调用一次，打印并返回拼接好的完整消息。
// 参数类型用 model.BaseChatModel 接口：任何实现了 Generate/Stream 的模型都能传进来。
func streamOnce(ctx context.Context, cm model.BaseChatModel, msgs []*schema.Message) (*schema.Message, error) {
	sr, err := cm.Stream(ctx, msgs)
	if err != nil {
		return nil, err
	}
	defer sr.Close()

	fmt.Print("AI：")
	var chunks []*schema.Message
	for {
		c, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		fmt.Print(c.Content)
		chunks = append(chunks, c)
	}
	return schema.ConcatMessages(chunks)
}
