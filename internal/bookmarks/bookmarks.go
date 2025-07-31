package bookmarks

import (
	"database/sql"
	"net/http"
	"net/url"
	"time"

	"github.com/segmentationfaulter/bookmarks-manager-api/internal/utils"
)

type Bookmark struct {
	Id          int
	Url         string
	Title       string
	Description string
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
