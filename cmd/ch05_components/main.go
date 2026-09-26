package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// ================= 3. 自定义 Retriever =================

// KeywordRetriever 是一个极简的“关键词检索器”。
// 它实现了 retriever.Retriever 接口：
//
//	Retrieve(ctx, query string, opts ...retriever.Option) ([]*schema.Document, error)
type KeywordRetriever struct {
	docs []*schema.Document
	topK int
}

// 编译期断言：确保 KeywordRetriever 实现了接口（没实现会编译失败）
var _ retriever.Retriever = (*KeywordRetriever)(nil)

func (r *KeywordRetriever) Retrieve(ctx context.Context,
	query string, opts ...retriever.Option) ([]*schema.Document, error) {
	// 解析通用选项：调用方可以用 retriever.WithTopK(n) 覆盖默认 TopK
	o := retriever.GetCommonOptions(&retriever.Options{TopK: &r.topK}, opts...)

	type scored struct {
		doc   *schema.Document
		score float64
	}

	var hits []scored
	for _, doc := range r.docs {
		// 按“查询里的每个字在文档中出现的次数”打分（仅用于演示）
		s := 0.0
		for _, ch := range query {
			s += float64(strings.Count(doc.Content, string(ch)))
		}

		if s > 0 {
			hits = append(hits, scored{doc: doc, score: s})
		}
	}

	sort.Slice(hits, func(i, j int) bool {
		return hits[i].score > hits[j].score
	})

	var out []*schema.Document
	for i := 0; i < len(hits) && i < *o.TopK; i++ {
		out = append(out, hits[i].doc.WithScore(hits[i].score)) // 分数写进 MetaData
	}
	return out, nil
}

func main() {
	ctx := context.Background()

	// ================= 1. Loader =================
	loader, err := file.NewFileLoader(ctx, &file.FileLoaderConfig{
		//UseNameAsID: true 表示使用文件名作为文档 ID。
		UseNameAsID: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	var docs []*schema.Document
	for _, p := range []string{"data/kb/test-env.md", "data/kb/release.md", "data/kb/go-style.md"} {
		ds, err := loader.Load(ctx, document.Source{
			URI: p,
		})
		if err != nil {
			log.Fatal(err)
		}
		docs = append(docs, ds...)
	}
	fmt.Printf("Loader：加载了 %d 个文档\n", len(docs))
	for _, d := range docs {
		fmt.Printf("  - ID=%s  长度=%d 字符  元数据=%v\n", d.ID, utf8.RuneCountInString(d.Content), keys(d.MetaData))
	}

	// ================= 2. Transformer（切分）=================
	splitter, err := recursive.NewSplitter(ctx, &recursive.Config{
		ChunkSize:   120, // 每块最大长度
		OverlapSize: 20,  // 相邻块重叠，避免语义被切断
		// 优先按二级标题切，其次段落、换行、句号
		Separators: []string{"\n## ", "\n\n", "\n", "。"},
		LenFunc:    utf8.RuneCountInString,
		IDGenerator: func(ctx context.Context, originalID string, splitIndex int) string {
			return fmt.Sprintf("%s#%d", originalID, splitIndex)
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	chunks, err := splitter.Transform(ctx, docs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nTransformer：切成了 %d 个片段，前 3 个：\n", len(chunks))
	for _, c := range chunks[:3] {
		fmt.Printf("	[%s] %q\n", c.ID, c.Content)
	}

	// ================= 3. 自定义 Retriever =================
	var r retriever.Retriever = &KeywordRetriever{
		docs: chunks,
		topK: 3,
	}
	results, err := r.Retrieve(ctx, "测试环境有效", retriever.WithTopK(2))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nRetriever：查询「测试环境有效期」")
	for _, d := range results {
		fmt.Printf("  score=%.0f [%s] %q\n", d.Score(), d.ID, d.Content)
	}

	// ================= 4. Lambda =================
	// InvokableLambda 把 func(ctx, I) (O, error) 包装成 *compose.Lambda，
	// 之后可以像组件一样放进 Chain / Graph（第 6 章）
	joinDocs := compose.InvokableLambda(func(ctx context.Context, ds []*schema.Document) (string, error) {
		var sb strings.Builder
		for i, d := range ds {
			fmt.Fprintf(&sb, "[%d] %s\n", i+1, d.Content)
		}
		return sb.String(), nil
	})
	fmt.Printf("\nLambda：%T 已创建（它还不能单独运行，要放进编排里）\n", joinDocs)
}

func keys(m map[string]any) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
