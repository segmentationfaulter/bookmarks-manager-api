package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/segmentationfaulter/bookmarks-manager-api/internal/bookmarks"
	"github.com/segmentationfaulter/bookmarks-manager-api/internal/tags"
	"github.com/segmentationfaulter/bookmarks-manager-api/internal/user"
	"github.com/segmentationfaulter/bookmarks-manager-api/internal/utils"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Unable to load environment variables: %v", err)
	}
	db, err := utils.InitDatabase()
	if err != nil {
		log.Fatalf("Unable to initialize database: %v", err)
	}
	defer db.Close()

	server := http.Server{
		Addr:    ":3000",
		Handler: Mux(db),
	}

	quitSignal := make(chan os.Signal, 1)
	signal.Notify(quitSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	<-quitSignal

	log.Println("Shutting down server...")

	// Give active requests up to 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly")
}

func Mux(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()

	// users endpoints
	mux.HandleFunc("POST /api/auth/register", user.RegisterationHandler(db))
	mux.HandleFunc("POST /api/auth/login", user.LoginHandler(db))
	mux.HandleFunc("GET /api/auth/me", user.ProfileHandler(db))

	// bookmarks endpoints
	mux.HandleFunc("POST /api/bookmarks", bookmarks.CreateBookmarkHandler(db))
	mux.HandleFunc("GET /api/bookmarks", bookmarks.GetBookmarksListHandler(db))
	mux.HandleFunc("GET /api/bookmarks/{id}", bookmarks.GetBookmarkHandler(db))
	mux.HandleFunc("PUT /api/bookmarks/{id}", bookmarks.UpdateBookmarkHandler(db))
	mux.HandleFunc("DELETE /api/bookmarks/{id}", bookmarks.DeleteBookmarkHandler(db))

	// tags endpoints
	mux.HandleFunc("GET /api/tags", tags.GetTagsHandler(db))
	mux.HandleFunc("DELETE /api/tags/{id}", tags.DeleteTagHandler(db))

	return mux
}
