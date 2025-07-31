package bookmarks

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/segmentationfaulter/bookmarks-manager-api/internal/utils"
)

type Bookmark struct {
	Id          int       `json:"id"`
	Url         string    `json:"url"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func GetBookmark(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := strconv.Atoi(id); err != nil || id == "" {
			http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
			return
		}

		userId, httpStatus, err := utils.IsAuthenticated(r)
		if err != nil {
			http.Error(w, err.Error(), httpStatus)
			return
		}
		bookmark, httpStatus, err := utils.FindOne(findBookmark(db, string(userId)), bookmarkScanner)
		if err != nil {
			http.Error(w, err.Error(), httpStatus)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bookmark)
	}
}

func CreateBookmark(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, httpStatus, err := utils.IsAuthenticated(r)
		if err != nil {
			http.Error(w, err.Error(), httpStatus)
			return
		}

		bookmark, err := utils.DecodeRequestBody[Bookmark](r)
		if err != nil {
			http.Error(w, "Error decoding request: "+err.Error(), http.StatusBadRequest)
			return
		}
		_, err = url.ParseRequestURI(bookmark.Url)
		if err != nil {
			http.Error(w, "Invalid URL: "+err.Error(), http.StatusBadRequest)
			return
		}

		if err := utils.Exec(db, utils.CREATE_BOOKMARK, userId, bookmark.Url, bookmark.Title, bookmark.Description, bookmark.Notes); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func findBookmark(db *sql.DB, id string) utils.QueryRunner {
	return func() (*sql.Row, error) {
		stmt, err := db.Prepare(utils.GET_BOOKMARK_BY_ID)

		if err != nil {
			return nil, err
		}
		return stmt.QueryRow(id), nil
	}
}

func bookmarkScanner(row *sql.Row) (*Bookmark, error) {
	bookmark := new(Bookmark)
	err := row.Scan(
		&bookmark.Id,
		&bookmark.Url,
		&bookmark.Title,
		&bookmark.Description,
		&bookmark.Notes,
		&bookmark.CreatedAt,
		&bookmark.UpdatedAt,
	)
	return bookmark, err
}
