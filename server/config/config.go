package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	ExpireDays int    `mapstructure:"expire_days"`
}

func (j JWTConfig) ExpireDuration() time.Duration {
	return time.Duration(j.ExpireDays) * 24 * time.Hour
}

type AIConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
	Enabled bool   `mapstructure:"enabled"`
	// Embedding 向量化模型配置, 供 AI 回顾(RAG)使用; 可与 chat 不同供应商
	Embedding EmbeddingConfig `mapstructure:"embedding"`
	// Recall AI 回顾行为配置, 设计见 doc/v0.3-tech-design.md
	Recall RecallConfig `mapstructure:"recall"`
}

// EmbeddingConfig OpenAI 兼容的 embedding 接口配置.
// Dims 必须与迁移 0009 中 vector 列的维度一致(启动时校验), 换维度需新迁移 + 全量重嵌.
// Asymmetric: 非对称模型(MiniMax embo-01)置 true, 文档走 type=db、查询走 type=query;
// 对称模型(OpenAI/SiliconFlow 等)保持 false.
// Style: "openai"(默认, 请求 input/响应 data[].embedding)或 "minimax"(请求 texts/
// 响应 vectors+base_resp)——实测 api.minimaxi.com 的 /embeddings 不认 OpenAI 格式.
type EmbeddingConfig struct {
	BaseURL    string `mapstructure:"base_url"`
	APIKey     string `mapstructure:"api_key"`
	Model      string `mapstructure:"model"`
	Dims       int    `mapstructure:"dims"`
	Asymmetric bool   `mapstructure:"asymmetric"`
	Style      string `mapstructure:"style"`
}

type RecallConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// TopK 每次对话最多召回的 chunk 数
	TopK int `mapstructure:"top_k"`
	// MinScore cosine 相似度下限, 低于则不注入任何上下文(闲聊行为不变)
	MinScore float64 `mapstructure:"min_score"`
	// IndexPrivate 是否索引私人笔记; 关闭时清除已索引的 private 块. draft 永不入索引.
	IndexPrivate bool `mapstructure:"index_private"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

// SiteConfig 站点级资源配置. IconDir 存自定义 favicon 全套文件,
// 必须位于前端 dist 部署路径之外, 避免被 deploy 覆盖.
type SiteConfig struct {
	IconDir string `mapstructure:"icon_dir"`
}

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	AI       AIConfig       `mapstructure:"ai"`
	CORS     CORSConfig     `mapstructure:"cors"`
	Site     SiteConfig     `mapstructure:"site"`
}

var globalCfg *Config

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("ZZDZZ")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if cfg.Site.IconDir == "" {
		cfg.Site.IconDir = "./data/icon"
	}

	globalCfg = &cfg
	return &cfg, nil
}

func Get() *Config {
	return globalCfg
}