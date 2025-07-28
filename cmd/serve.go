package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/segmentationfaulter/bookmarks-manager-api/internal/db"
	"github.com/segmentationfaulter/bookmarks-manager-api/internal/user"
)

func main() {
	db, err := db.InitDatabase()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(http.ListenAndServe(":3000", Mux(db)))
}

func Mux(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		user, err := user.ParseUser(w, r)
		if err != nil {
			http.Error(w, "Error decoding request body", http.StatusBadRequest)
			return
		}

		err = user.Save(db)
		if err != nil {
			http.Error(w, "Error saving user to DB", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
	return mux
}
