package main

import (
	"encoding/json"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
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
		err := json.NewDecoder(r.Body).Decode(&regData)
		if err != nil {
			http.Error(w, "Error decoding request body", http.StatusBadRequest)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(regData.Password), bcrypt.DefaultCost)

		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}
	})
	return mux
}
