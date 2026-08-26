// Package config 负责加载服务运行配置与环境变量。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config 保存服务运行所需的全部配置项。
type Config struct {
	// Port 服务监听端口，默认 8080，支持 PORT 环境变量覆盖。
	Port string
	// DataFile JSON 持久化文件路径，默认 data/store.json。
	DataFile string
	// LogLevel 结构化日志级别，支持 debug/info/warn/error。
	LogLevel string
	// HTTP 服务超时与优雅关闭配置，支持环境变量覆盖。
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	// Rules 业务规则（灭菌参数下限、有效期、回收时限等）。
	Rules Rules
}

// Load 从环境变量与默认值构建配置。非法环境变量值回退到默认值，
// 最终合法性由 Validate 统一校验。
func Load() Config {
	cfg := Config{
		Port:              envOr("PORT", "8080"),
		DataFile:          envOr("DATA_FILE", filepath.Join("data", "store.json")),
		LogLevel:          strings.ToUpper(strings.TrimSpace(envOr("LOG_LEVEL", "info"))),
		ReadTimeout:       envDuration("HTTP_READ_TIMEOUT", 15*time.Second),
		ReadHeaderTimeout: envDuration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
		WriteTimeout:      envDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:       envDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout:   envDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		Rules:             DefaultRules(),
	}
	cfg.Rules = LoadRulesFromEnv(cfg.Rules)
	return cfg
}

// Validate 校验最终配置。运行期环境变量非法时由 main 直接拒绝启动，
// 避免在错误配置下静默降级。
func (c Config) Validate() error {
	if n, err := strconv.Atoi(c.Port); err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("PORT 必须是 1-65535 的整数: %q", c.Port)
	}
	if strings.TrimSpace(c.DataFile) == "" {
		return fmt.Errorf("DATA_FILE 不能为空")
	}
	switch c.LogLevel {
	case "DEBUG", "INFO", "WARN", "ERROR":
	default:
		return fmt.Errorf("LOG_LEVEL 必须是 debug/info/warn/error: %q", c.LogLevel)
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("HTTP_READ_TIMEOUT 必须大于 0")
	}
	if c.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("HTTP_READ_HEADER_TIMEOUT 必须大于 0")
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("HTTP_WRITE_TIMEOUT 必须大于 0")
	}
	if c.IdleTimeout <= 0 {
		return fmt.Errorf("HTTP_IDLE_TIMEOUT 必须大于 0")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT 必须大于 0")
	}
	return c.Rules.Validate()
}

// envOr 读取环境变量，若为空则返回 fallback。
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envInt 读取整型环境变量，解析失败时返回 fallback。
func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// envFloat 读取浮点环境变量，解析失败时返回 fallback。
func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

// envDuration 读取 Go duration 环境变量，解析失败时返回 fallback。
func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
