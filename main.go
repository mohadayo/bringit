package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

// securityHeaders はセキュリティ関連のHTTPヘッダを付与するミドルウェア。
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; form-action 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// buildShareURL はリクエスト情報からシェア用の絶対URLを組み立てる。
// X-Forwarded-Proto ヘッダがあればそれを使い、なければデフォルトで http を使用する。
func buildShareURL(r *http.Request, token string) string {
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/lists/" + token
}

func main() {
	store := NewStore(
		getEnvInt("MAX_LISTS", 1000),
		getEnvInt("MAX_ITEMS_PER_LIST", 200),
	)
	mux := http.NewServeMux()
	registerRoutes(mux, store)

	rl := newRateLimiter(getEnvInt("RATE_LIMIT", 60), time.Minute)
	rl.startCleanup()

	addr := ":" + getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           securityHeaders(rateLimitMiddleware(rl)(requestLogger(mux))),
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
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

