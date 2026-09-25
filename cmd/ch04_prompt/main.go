package main

import (
	"context"
	"eino-tutorial/internal/config"
	"eino-tutorial/internal/llm"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ========== 1. FString 格式（Python 风格 {变量}）==========
	// prompt.FromMessages(格式, 消息模板...) 返回 *prompt.DefaultChatTemplate，
	// 它实现了 prompt.ChatTemplate 接口：Format(ctx, map[string]any) ([]*schema.Message, error)
	tpl := prompt.FromMessages(schema.FString,
		// System Prompt：定义角色、规则、输出格式
		schema.SystemMessage(
			"你是{company}的{role}。\n"+
				"规则：\n1. 只回答与{domain}相关的问题\n2. 回答不超过{max_words}字\n"+
				"3. 如果需要输出 JSON，示例：{{\"ok\": true}}"), // 字面量花括号要写成 {{ }})
		// 历史消息占位符：运行时用 []*schema.Message 替换；true 表示可以不传
		schema.MessagesPlaceholder("history", true),
		// User Prompt：用户问题
		schema.UserMessage("问题：{question}"),
	)

	history := []*schema.Message{
		schema.UserMessage("我们用的是 Go 1.25"),
		schema.AssistantMessage("好的，我会基于 Go 1.25 回答。", nil),
	}

	vars := map[string]any{
		"company":   "ROTECH",
		"role":      "Go 技术顾问",
		"domain":    "Go 后端开发",
		"max_words": 150,
		"history":   history, // 占位符的值必须是 []*schema.Message
		"question":  "for range 循环变量在 Go 1.22 之后有什么变化？",
	}

	msgs, err := tpl.Format(ctx, vars)
	if err != nil {
		log.Fatal("Format失败：", err)
	}

	fmt.Println("===== 渲染结果（FString）=====")
	for _, m := range msgs {
		fmt.Printf("[%s] %s\n", m.Role, m.Content)
	}

	// ========== 2. GoTemplate 格式（text/template 语法，支持 if / range）==========
	goTpl := prompt.FromMessages(schema.GoTemplate,
		schema.SystemMessage(`你是代码审查员。{{if .strict}}请严格审查，任何风格问题都要指出。{{else}}只关注严重问题。{{end}}`),
		schema.UserMessage(`请审查以下文件：
			{{range .files}}- {{.}}
			{{end}}`),
	)

	msgs2, err := goTpl.Format(ctx, map[string]any{
		"strict": true,
		"files":  []string{"user.go", "order.go"},
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n===== 渲染结果（GoTemplate）=====")
	for _, m := range msgs2 {
		fmt.Printf("[%s] %s\n", m.Role, m.Content)
	}

	// ========== 3. Jinja2 格式 ==========
	jTpl := prompt.FromMessages(schema.Jinja2,
		schema.UserMessage(`你好 {{ name }}，你有 {{ items | length }} 个待办：
{% for i in items %}{{ i }}；{% endfor %}`),
	)

	msgs3, err := jTpl.Format(ctx, map[string]any{
		"name":  "Tian",
		"items": []any{"写周报", "修 bug"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\n===== 渲染结果（Jinja2）=====")
	fmt.Println(msgs3[0].Content)

	// ========== 4. 手动组合：Template → ChatModel ==========
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	cm, err := llm.NewChatModel(ctx2, cfg.LLM, llm.Options{
		DisableThinking: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := cm.Generate(ctx2, msgs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\n===== 模型回答 =====\n" + resp.Content)
}
