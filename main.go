package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	store := NewStore()
	mux := http.NewServeMux()
	registerRoutes(mux, store)

	addr := ":8080"
	fmt.Println("BringIt server started on http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
