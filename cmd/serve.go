package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	log.Fatal(http.ListenAndServe(":3000", Mux()))
}

func Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Add("content-type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Message string `json:"message"`
		}{Message: "Registeration successful"})
	})

	return mux
}
