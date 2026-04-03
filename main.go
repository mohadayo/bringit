package main

import (
	"log"
	"net/http"
	"os"
)

// loggingMiddleware logs each incoming HTTP request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request: %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func main() {
	store := NewStore()
	mux := http.NewServeMux()
	registerRoutes(mux, store)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	handler := loggingMiddleware(mux)

	log.Printf("BringIt server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
