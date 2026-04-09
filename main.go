package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// getEnv は環境変数を取得し、未設定の場合はデフォルト値を返す。
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func main() {
	store := NewStore()
	mux := http.NewServeMux()
	registerRoutes(mux, store)

	addr := ":" + getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:         addr,
		Handler:      requestLogger(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// シグナルを受信してグレースフルシャットダウンするチャネル
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("BringItサーバーを起動しました", "addr", "http://localhost"+addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("サーバーの起動に失敗しました", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("シャットダウンシグナルを受信しました")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("グレースフルシャットダウンに失敗しました", "error", err)
		os.Exit(1)
	}
	slog.Info("サーバーを正常に停止しました")
}
