// 消毒供应中心器械追溯服务（cssd-instrument-trace-service）
// 基于 Go 实现的消毒供应中心器械追溯 Web 项目，一款后端服务，
// 完成器械包登记、清洗消毒灭菌批次管理、发放回收闭环、灭菌参数合格判定与追溯查询。
package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"example.com/cssd-instrument-trace-service/config"
	"example.com/cssd-instrument-trace-service/httpapi"
	"example.com/cssd-instrument-trace-service/service"
	"example.com/cssd-instrument-trace-service/store"
)

//go:embed all:web
var embeddedWeb embed.FS

func main() {
	if err := run(); err != nil {
		slog.Error("服务启动失败", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return err
	}
	configureLogger(cfg.LogLevel)

	st, err := store.New(cfg.DataFile)
	if err != nil {
		return err
	}
	svc := service.New(st, cfg.Rules)

	webFS, err := fs.Sub(embeddedWeb, "web")
	if err != nil {
		return err
	}

	handler := httpapi.Router(svc, webFS)
	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	// 启动过期扫描定时任务。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sweeper := service.NewExpirySweeper(svc, cfg.Rules.SweepIntervalMin)
	go sweeper.Run(ctx)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("服务已启动",
			"addr", addr,
			"data_file", cfg.DataFile,
			"log_level", strings.ToLower(cfg.LogLevel),
		)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case <-done:
		slog.Info("收到退出信号，正在关闭服务")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("服务关闭异常", "error", err)
		}
		slog.Info("服务已关闭")
		return nil
	case err := <-serverErr:
		cancel()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

// configureLogger 设置全局结构化日志级别。LOG_LEVEL 已在 config 校验，
// 这里仍保留防御性回退。
func configureLogger(level string) {
	var lvl slog.Level
	switch strings.ToUpper(level) {
	case "DEBUG":
		lvl = slog.LevelDebug
	case "WARN":
		lvl = slog.LevelWarn
	case "ERROR":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})))
}
