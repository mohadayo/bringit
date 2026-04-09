package main

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// responseWriter はステータスコードを記録するためのラッパー。
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// requestLogger はリクエストのメソッド・パス・ステータス・所要時間をログ出力するミドルウェア。
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		slog.Info("リクエスト処理",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration", time.Since(start).String(),
			"remote", r.RemoteAddr,
		)
	})
}

// startTime はサーバー起動時刻を記録する。
var startTime = time.Now()

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"progress": func(items []*Item) int {
		if len(items) == 0 {
			return 0
		}
		done := 0
		for _, it := range items {
			if it.Prepared {
				done++
			}
		}
		return done * 100 / len(items)
	},
}).ParseGlob("templates/*.html"))

func registerRoutes(mux *http.ServeMux, store *Store) {
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("POST /lists", handleCreateList(store))
	mux.HandleFunc("GET /lists/{token}", handleShowList(store))
	mux.HandleFunc("POST /lists/{token}/items", handleAddItem(store))
	mux.HandleFunc("POST /lists/{token}/items/{id}/toggle-prepared", handleTogglePrepared(store))
	mux.HandleFunc("POST /lists/{token}/items/{id}/toggle-required", handleToggleRequired(store))
	mux.HandleFunc("POST /lists/{token}/items/{id}/assignee", handleUpdateAssignee(store))
	mux.HandleFunc("POST /lists/{token}/items/{id}/delete", handleDeleteItem(store))
	mux.HandleFunc("POST /lists/{token}/delete", handleDeleteList(store))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
		"status":    "ok",
		"service":   "bringit",
		"uptime":    time.Since(startTime).Round(time.Second).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		slog.Error("テンプレート描画エラー", "template", "index.html", "error", err)
		http.Error(w, "テンプレートの描画に失敗しました", http.StatusInternalServerError)
	}
}

func handleCreateList(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "タイトルは必須です", http.StatusBadRequest)
			return
		}
		desc := strings.TrimSpace(r.FormValue("description"))
		l := store.CreateList(title, desc)
		http.Redirect(w, r, "/lists/"+l.ShareToken, http.StatusSeeOther)
	}
}

func handleShowList(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		l := store.GetList(token)
		if l == nil {
			http.NotFound(w, r)
			return
		}
		data := map[string]any{
			"List":     l,
			"ShareURL": "http://" + r.Host + "/lists/" + l.ShareToken,
		}
		if err := tmpl.ExecuteTemplate(w, "list.html", data); err != nil {
			slog.Error("テンプレート描画エラー", "template", "list.html", "error", err)
			http.Error(w, "テンプレートの描画に失敗しました", http.StatusInternalServerError)
		}
	}
}

func handleAddItem(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			http.Redirect(w, r, "/lists/"+token, http.StatusSeeOther)
			return
		}
		assignee := strings.TrimSpace(r.FormValue("assignee"))
		required := r.FormValue("required") == "on"
		store.AddItem(token, name, assignee, required)
		http.Redirect(w, r, "/lists/"+token, http.StatusSeeOther)
	}
}

func handleTogglePrepared(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		id := r.PathValue("id")
		store.TogglePrepared(token, id)
		http.Redirect(w, r, "/lists/"+token, http.StatusSeeOther)
	}
}

func handleToggleRequired(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		id := r.PathValue("id")
		store.ToggleRequired(token, id)
		http.Redirect(w, r, "/lists/"+token, http.StatusSeeOther)
	}
}

func handleUpdateAssignee(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		id := r.PathValue("id")
		assignee := strings.TrimSpace(r.FormValue("assignee"))
		store.UpdateAssignee(token, id, assignee)
		http.Redirect(w, r, "/lists/"+token, http.StatusSeeOther)
	}
}

func handleDeleteItem(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		id := r.PathValue("id")
		store.DeleteItem(token, id)
		http.Redirect(w, r, "/lists/"+token, http.StatusSeeOther)
	}
}

func handleDeleteList(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		if !store.DeleteList(token) {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
