package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

// TranslateReq 是整条链的输入（业务侧的强类型结构体）
type TranslateReq struct {
	Text   string
	Target string // 目标语言
	Style  string // 风格：正式 / 口语
}

// TranslateResp 是整条链的输出
type TranslateResp struct {
	Result string
	Tokens int
}

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	_ = godotenv.Load()

	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL:     os.Getenv("DEEPSEEK_BASE_URL"), // DeepSeek: https://api.deepseek.com
		Model:       os.Getenv("DEEPSEEK_MODEL"),
		Temperature: new(float32(0.3)),
	})

	if err != nil {
		log.Fatal(err)
	}

	// 节点 1（Lambda）：TranslateReq → map[string]any（模板需要的变量）
	//InvokableLambda 的作用，就是把这个普通 Go 函数包装成能加入 Chain 的节点。
	//type InvokeWOOpt[I any, O any] func(ctx context.Context, input I) (output O, err error)
	//InvokableLambda[I any, O any](i InvokeWOOpt[I, O], opts ...LambdaOpt) *Lambda
	toVars := compose.InvokableLambda(func(ctx context.Context, in *TranslateReq) (map[string]any, error) {
		if strings.TrimSpace(in.Text) == "" {
			return nil, errors.New("empty text")
		}
		style := in.Style
		if style == "" {
			style = "正式"
		}
		return map[string]any{"text": in.Text, "target": in.Target, "style": style}, nil
	})

	// 节点 2（ChatTemplate）：map[string]any → []*schema.Message
	tpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是专业翻译。把用户给的文本翻译成{target}，"+
			"风格：{style}。只输出译文，不要解释。"),
		schema.UserMessage("{text}"),
	)

	// 节点 3（ChatModel）：[]*schema.Message → *schema.Message
	// 节点 4（Lambda）：*schema.Message → *TranslateResp
	toResp := compose.InvokableLambda(func(ctx context.Context, m *schema.Message) (*TranslateResp, error) {
		r := &TranslateResp{Result: strings.TrimSpace(m.Content)}
		if m.ResponseMeta != nil && m.ResponseMeta.Usage != nil {
			r.Tokens = m.ResponseMeta.Usage.TotalTokens
		}
		return r, nil
	})

	// NewChain[I, O]：I 是整条链的输入类型，O 是输出类型
	chain := compose.NewChain[*TranslateReq, *TranslateResp]()

	//把业务请求转换为模板变量
	chain.AppendLambda(toVars, compose.WithNodeName("to_vars")).
		//把变量填入提示词
		AppendChatTemplate(tpl, compose.WithNodeName("prompt")).
		//调用模型生成翻译
		AppendChatModel(cm, compose.WithNodeName("model")).
		//整理成业务结果
		AppendLambda(toResp, compose.WithNodeName("to_resp"))

	// Compile：检查相邻节点类型是否匹配，并生成可执行的 Runnable。
	// Runnable 是并发安全的，应该在程序启动时编译一次，之后反复使用。
	runnable, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal("compile fail", err)
	}

	// ---------- Invoke：非流式 ----------
	out, err := runnable.Invoke(ctx, &TranslateReq{
		Text:   "今天的发布推迟到周四下午。",
		Target: "中文",
	})

	if err != nil {
		log.Fatal("execute fail", err)
	}

	fmt.Printf("Invoke 结果：%s（tokens=%d）\n", out.Result, out.Tokens)

	// ---------- chain2：以 ChatModel 结尾，可以逐字流式输出 ----------
	//直接输入变量，直接输出模型消息
	chain2 := compose.NewChain[map[string]any, *schema.Message]()
	chain2.AppendChatTemplate(tpl).AppendChatModel(cm)
	r2, err := chain2.Compile(ctx)
	if err != nil {
		log.Fatal("compile fail", err)
	}

	out2, err := r2.Invoke(ctx, map[string]any{
		"text":   "编译一次，到处运行",
		"target": "英文",
		"style":  "口语",
	})

	if err != nil {
		log.Fatal("execute fail", err)
	}
	fmt.Printf("r2 invoke message: %s\n", out2.Content)

	sr2, err := r2.Stream(ctx, map[string]any{
		"text":   "编译一次，到处运行",
		"target": "英文",
		"style":  "口语",
	})
	if err != nil {
		log.Fatal("stream fail", err)
	}
	fmt.Print("chain2 逐字输出：")
	for {
		m, err := sr2.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatal("receive fail", err)
		}
		fmt.Print(m.Content)
	}

	sr2.Close()
	fmt.Println()
}
