package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	log.Fatal(http.ListenAndServe(":3000", Mux()))
}

type RegisterationData struct {
	Username string
	Email    string
	Password string
}

func Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		var regData = RegisterationData{}
		json.NewDecoder(r.Body).Decode(&regData)
		fmt.Printf("username: %v\nemail: %v\n", regData.Username, regData.Email)
	})

	return mux
}
