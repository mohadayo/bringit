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

// securityHeaders はセキュリティ関連のHTTPヘッダを付与するミドルウェア。
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
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
	store := NewStore()
	mux := http.NewServeMux()
	registerRoutes(mux, store)

	addr := ":" + getEnv("PORT", "8080")
	slog.Info("BringItサーバーを起動しました", "addr", "http://localhost"+addr)
	if err := http.ListenAndServe(addr, securityHeaders(mux)); err != nil {
		slog.Error("サーバーの起動に失敗しました", "error", err)
		os.Exit(1)
	}
}
