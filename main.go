package main

import (
	"log/slog"
	"net/http"
	"os"
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
	slog.Info("BringItサーバーを起動しました", "addr", "http://localhost"+addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("サーバーの起動に失敗しました", "error", err)
		os.Exit(1)
	}
}
