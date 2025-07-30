package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/segmentationfaulter/bookmarks-manager-api/internal/db"
	"github.com/segmentationfaulter/bookmarks-manager-api/internal/user"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Unable to load environment variables")
	}
	db, err := db.InitDatabase()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(http.ListenAndServe(":3000", Mux(db)))
}

func Mux(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", user.RegisterationHandler(db))
	mux.HandleFunc("POST /api/auth/login", user.LoginHandler(db))
	mux.HandleFunc("GET /api/auth/me", user.ProfileHandler(db))
	return mux
}
