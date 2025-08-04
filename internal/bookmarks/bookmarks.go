package bookmarks

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/segmentationfaulter/bookmarks-manager-api/internal/tags"
	"github.com/segmentationfaulter/bookmarks-manager-api/internal/utils"
)

type BookmarksQueryParams struct {
	page   int
	limit  int
	tags   []string
	search string
	sort   string
	order  string
}

type Bookmark struct {
	Id          int       `json:"id"`
	Url         string    `json:"url"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BookmarkWithTag struct {
	Bookmark
	Tag string `json:"tag"`
}

type BookmarkWithTags struct {
	Bookmark
	Tags []string `json:"tags"`
}

func GetBookmarksList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, httpStatus, err := utils.IsAuthenticated(r)
		if err != nil {
			http.Error(w, err.Error(), httpStatus)
			return
		}
		queryParams := getQueryParams(r)
		bookmarks, err := bookmarksList(db, string(userId), queryParams)
		if err != nil {
			http.Error(w, "Error getting bookmarks: "+err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(normalizeBookmarks(bookmarks))
	}
}

// TODO: We need to include tags here
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
		bookmark, httpStatus, err := utils.FindOne(findBookmark(db, id, string(userId)), bookmarkScanner)
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

		bookmark, err := utils.DecodeRequestBody[struct {
			Bookmark
			Tags []string
		}](r)
		if err != nil {
			http.Error(w, "Error decoding request: "+err.Error(), http.StatusBadRequest)
			return
		}
		_, err = url.ParseRequestURI(bookmark.Url)
		if err != nil {
			http.Error(w, "Invalid URL: "+err.Error(), http.StatusBadRequest)
			return
		}

		tx, err := db.Begin()

		if err != nil {
			tx.Rollback()
			http.Error(w, "Couldn't start transaction: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tags.CreateTags(tx, bookmark.Tags, string(userId)); err != nil {
			tx.Rollback()
			http.Error(w, "Error creating tags: "+err.Error(), http.StatusInternalServerError)
			return
		}

		bookmarksExecResult, err := utils.Exec(tx, utils.CREATE_BOOKMARK, userId, bookmark.Url, bookmark.Title, bookmark.Description, bookmark.Notes)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Error creating bookmark"+err.Error(), http.StatusInternalServerError)
			return
		}

		bookmarkId, err := bookmarksExecResult.LastInsertId()
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		savedTags, err := tags.GetTags(tx, bookmark.Tags, string(userId))
		if err != nil {
			tx.Rollback()
			http.Error(w, "Error getting tags Ids"+err.Error(), http.StatusInternalServerError)
			return
		}

		err = tags.UpdateBookmarkTags(tx, bookmarkId, tags.TagIds(savedTags))
		if err != nil {
			tx.Rollback()
			http.Error(w, "Error updating boookmark_tags table"+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func findBookmark(db *sql.DB, id string, userId string) utils.QueryRunner {
	return func() (*sql.Row, error) {
		stmt, err := db.Prepare(utils.GET_BOOKMARK_BY_ID)

		if err != nil {
			return nil, err
		}
		return stmt.QueryRow(id, userId), nil
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

func bookmarksListQuery(userID string, queryParams BookmarksQueryParams) string {
	var search string
	if queryParams.search == "" {
		search = "1=1"
	} else {
		search = "title LIKE ? OR description LIKE ? OR notes LIKE ?"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT b.id, b.url, b.title, b.description, b.notes, b.created_at, b.updated_at, t.name
		FROM bookmarks b
		LEFT JOIN bookmark_tags b_t
		ON b.id = b_t.bookmark_id
		LEFT JOIN tags t
		ON b_t.tag_id = t.id
		WHERE b.user_id = %s
		  AND (%s)
		ORDER BY b.%s %s
		LIMIT ? OFFSET ?;`,
		userID, search, queryParams.sort, queryParams.order)
	return query
}

func normalizeBookmarks(bookmarks []BookmarkWithTag) []BookmarkWithTags {
	var result []BookmarkWithTags

	for _, bookmark := range bookmarks {
		indexOfExistingBookmark := slices.IndexFunc(result, func(e BookmarkWithTags) bool {
			return e.Id == bookmark.Id
		})

		if indexOfExistingBookmark < 0 {
			if bookmark.Tag == "" {
				result = append(result, BookmarkWithTags{
					Bookmark: bookmark.Bookmark,
					Tags:     nil,
				})
			} else {
				result = append(result, BookmarkWithTags{
					Bookmark: bookmark.Bookmark,
					Tags:     []string{bookmark.Tag},
				})
			}

		} else {
			result[indexOfExistingBookmark].Tags = append(result[indexOfExistingBookmark].Tags, bookmark.Tag)
		}
	}

	return result
}

func bookmarksList(db *sql.DB, userID string, queryParams BookmarksQueryParams) ([]BookmarkWithTag, error) {
	search := queryParams.search
	query := bookmarksListQuery(userID, queryParams)

	args := []any{}
	if search != "" {
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	args = append(args, queryParams.limit, queryParams.limit*(queryParams.page-1))

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tag sql.NullString
	var result []BookmarkWithTag

	for rows.Next() {
		bookmark := BookmarkWithTag{}
		if err := rows.Scan(
			&bookmark.Id,
			&bookmark.Url,
			&bookmark.Title,
			&bookmark.Description,
			&bookmark.Notes,
			&bookmark.CreatedAt,
			&bookmark.UpdatedAt,
			&tag,
		); err != nil {
			return nil, err
		}

		if tag.Valid {
			bookmark.Tag = tag.String
		}
		result = append(result, bookmark)
	}

	return result, rows.Err()
}

func getQueryParams(r *http.Request) BookmarksQueryParams {
	defaultParams := BookmarksQueryParams{
		page:  1,
		limit: 20,
		sort:  "created_at",
		order: "desc",
		tags:  []string{},
	}
	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil {
		defaultParams.page = page
	}

	if limit, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		defaultParams.limit = limit
	}

	if tags := strings.Split(r.URL.Query().Get("tags"), ","); len(tags) > 0 {
		defaultParams.tags = tags
	}

	defaultParams.search = r.URL.Query().Get("search")

	if sort := r.URL.Query().Get("sort"); sort == "updated_at" || sort == "title" || sort == "url" {
		defaultParams.sort = sort
	}

	if order := r.URL.Query().Get("order"); order == "asc" {
		defaultParams.order = order
	}

	return defaultParams
}
