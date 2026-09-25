package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type LLM struct {
	Provider string // deepseek / deepseek-native / openai / gemini
	APIKey   string
	BaseURL  string
	Model    string
}

// Embedding 描述向量模型的接入信息
type Embedding struct {
	APIKey  string
	BaseURL string
	Model   string
	Dim     int
}

type Config struct {
	LLM       LLM
	ProModel  string // 可选：更强的模型名，用于 Review 等场景
	Embedding Embedding
	RedisAddr string
	RedisPwd  string
	MySQLDSN  string
}

// Load 读取配置。
// 会先尝试加载当前目录的 .env 文件（不存在也不报错），再读取环境变量。
// 真实环境变量的优先级高于 .env（godotenv.Load 不会覆盖已存在的环境变量）。
func Load() (*Config, error) {
	_ = godotenv.Load()

	c := &Config{
		RedisAddr: getenv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPwd:  os.Getenv("REDIS_PASSWORD"),
		MySQLDSN:  os.Getenv("MYSQL_DSN"),
	}

	c.LLM.Provider = getenv("LLM_PROVIDER", "deepseek")
	switch c.LLM.Provider {
	case "deepseek", "deepseek-native":
		c.LLM.APIKey = os.Getenv("DEEPSEEK_API_KEY")
		c.LLM.BaseURL = getenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com")
		c.LLM.Model = os.Getenv("DEEPSEEK_MODEL")
		c.ProModel = getenv("DEEPSEEK_PRO_MODEL", c.LLM.Model)
	case "openai":
		c.LLM.APIKey = os.Getenv("OPENAI_API_KEY")
		c.LLM.BaseURL = getenv("OPENAI_BASE_URL", "https://api.openai.com/v1")
		c.LLM.Model = os.Getenv("OPENAI_MODEL")
		c.ProModel = c.LLM.Model
	case "gemini":
		c.LLM.APIKey = os.Getenv("GEMINI_API_KEY")
		c.LLM.Model = os.Getenv("GEMINI_MODEL")
		c.ProModel = c.LLM.Model
	default:
		return nil, fmt.Errorf("未知的 LLM_PROVIDER: %s", c.LLM.Provider)
	}
	if c.LLM.APIKey == "" || c.LLM.Model == "" {
		return nil, fmt.Errorf("provider=%s 的 API Key 或 Model 未配置", c.LLM.Provider)
	}

	c.Embedding = Embedding{
		APIKey:  os.Getenv("EMBEDDING_API_KEY"),
		BaseURL: os.Getenv("EMBEDDING_BASE_URL"),
		Model:   os.Getenv("EMBEDDING_MODEL"),
		Dim:     getenvInt("EMBEDDING_DIM", 1024),
	}
	return c, nil
}

// MustLoad 用于示例程序：配置错误直接退出
func MustLoad() *Config {
	c, err := Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "配置错误:", err)
		os.Exit(1)
	}
	return c
}

func getenv(key string, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
