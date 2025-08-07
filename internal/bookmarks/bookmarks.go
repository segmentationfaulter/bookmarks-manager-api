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
		bookmarks, err := utils.FindMany(
			bookmarksListQueryRunner(db, string(userId), queryParams),
			bookmarksScanner,
		)
		if err != nil {
			http.Error(w, "Error getting bookmarks: "+err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(normalizeBookmarks(bookmarks))
	}
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

		bookmarks, err := utils.FindMany(
			bookmarkByIdQueryRunner(db, string(userId), id),
			bookmarksScanner,
		)
		if err != nil {
			http.Error(w, "Error getting bookmark "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(normalizeBookmarks(bookmarks))
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
			http.Error(w, "Error getting tags Ids: "+err.Error(), http.StatusInternalServerError)
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

func UpdateBookmark(db *sql.DB) http.HandlerFunc {
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

		existingBookmark, httpStatus, err := utils.FindOne(findBookmark(db, id, string(userId)), bookmarkScanner)
		if err != nil {
			http.Error(w, err.Error(), httpStatus)
			return
		}

		newBookmark, err := utils.DecodeRequestBody[struct {
			Bookmark
			Tags []string
		}](r)
		if err != nil {
			http.Error(w, "Error decoding request: "+err.Error(), http.StatusBadRequest)
			return
		}

		_, err = url.ParseRequestURI(newBookmark.Url)
		if newBookmark.Url != "" && err != nil {
			http.Error(w, "Invalid URL: "+err.Error(), http.StatusBadRequest)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			tx.Rollback()
			http.Error(w, "Couldn't start transaction: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := execUpdateBookmark(tx, string(userId), *existingBookmark, newBookmark.Bookmark); err != nil {
			tx.Rollback()
			http.Error(w, "Bookmark update failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := deleteBookmarkTagIds(tx, id); err != nil {
			tx.Rollback()
			http.Error(w, "Couldn't delete existing mappings in junction table: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tags.CreateTags(tx, newBookmark.Tags, string(userId)); err != nil {
			tx.Rollback()
			http.Error(w, "Error creating new tags: "+err.Error(), http.StatusInternalServerError)
			return
		}

		savedTags, err := tags.GetTags(tx, newBookmark.Tags, string(userId))
		if err != nil {
			tx.Rollback()
			http.Error(w, "Error getting tags Ids: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = tags.UpdateBookmarkTags(tx, int64(existingBookmark.Id), tags.TagIds(savedTags))
		if err != nil {
			tx.Rollback()
			http.Error(w, "Error updating boookmark_tags junction table"+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func bookmarksListQuery(userID string, queryParams BookmarksQueryParams) string {
	var search string
	var tagsQuery string

	if len(queryParams.tags) > 0 {
		placeholders := []string{}
		for _, _ = range queryParams.tags {
			placeholders = append(placeholders, "?")
		}
		tagsQuery = "t.name IN(" + strings.Join(placeholders, ", ") + ")"
	} else {
		tagsQuery = "1=1"
	}

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
		  AND %s
		  AND %s
		ORDER BY b.%s %s
		LIMIT ? OFFSET ?;`,
		userID, search, tagsQuery, queryParams.sort, queryParams.order)

	return query
}

func bookmarkByIdQueryRunner(db *sql.DB, userId, bookmarkId string) func() (*sql.Stmt, *sql.Rows, error) {
	return func() (*sql.Stmt, *sql.Rows, error) {
		query := `
			SELECT b.id, b.url, b.title, b.description, b.notes, b.created_at, b.updated_at, t.name
			FROM bookmarks b
			LEFT JOIN bookmark_tags b_t
			ON b.id = b_t.bookmark_id
			LEFT JOIN tags t
			ON b_t.tag_id = t.id
			WHERE b.user_id = ?
			  AND b.id = ?
		`

		stmt, err := db.Prepare(query)
		if err != nil {
			return nil, nil, err
		}

		rows, err := db.Query(query, userId, bookmarkId)
		if err != nil {
			return stmt, nil, err
		}

		return stmt, rows, err
	}
}

func bookmarksScanner(rows *sql.Rows) ([]BookmarkWithTag, error) {
	var result []BookmarkWithTag

	for rows.Next() {
		var bookmark BookmarkWithTag
		var tag sql.NullString
		err := rows.Scan(
			&bookmark.Id,
			&bookmark.Url,
			&bookmark.Title,
			&bookmark.Description,
			&bookmark.Notes,
			&bookmark.CreatedAt,
			&bookmark.UpdatedAt,
			&tag,
		)

		if err != nil {
			return nil, err
		}

		if tag.Valid {
			bookmark.Tag = tag.String
		}

		result = append(result, bookmark)
	}

	return result, nil
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

func bookmarksListQueryRunner(
	db *sql.DB, userId string,
	queryParams BookmarksQueryParams,
) func() (*sql.Stmt, *sql.Rows, error) {
	return func() (*sql.Stmt, *sql.Rows, error) {
		search := queryParams.search
		query := bookmarksListQuery(userId, queryParams)

		args := []any{}
		if search != "" {
			args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
		}
		if len(queryParams.tags) > 0 {
			for _, tag := range queryParams.tags {
				args = append(args, tag)
			}
		}
		args = append(args, queryParams.limit, queryParams.limit*(queryParams.page-1))

		stmt, err := db.Prepare(query)
		if err != nil {
			return nil, nil, err
		}

		rows, err := stmt.Query(args...)

		if err != nil {
			return stmt, nil, err
		}

		return stmt, rows, nil
	}
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

	if tags := r.URL.Query().Get("tags"); len(tags) > 0 {
		defaultParams.tags = strings.Split(tags, ",")
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

func execUpdateBookmark(
	execer utils.Execer,
	userId string,
	existingBookmark Bookmark,
	newBookmark Bookmark,
) error {
	var url, title, description, notes *string

	if newBookmark.Url == "" {
		url = nil
	} else {
		url = &existingBookmark.Url
	}

	if newBookmark.Title == "" {
		title = nil
	} else {
		title = &existingBookmark.Title
	}

	if newBookmark.Description == "" {
		description = nil
	} else {
		description = &existingBookmark.Description
	}

	if newBookmark.Notes == "" {
		notes = nil
	} else {
		notes = &existingBookmark.Notes
	}

	query := fmt.Sprintf(`
			UPDATE bookmarks
			SET
				url = COALESCE(?, '%v'),
				title = COALESCE(?, '%s'),
				description = COALESCE(?, '%s'),
				notes = COALESCE(?, '%s')
			WHERE
				id = %d
			AND user_id = %s;
		`, *url, *title, *description, *notes, existingBookmark.Id, userId)

	_, err := utils.Exec(execer, query, url, title, description, notes)
	return err
}

func findBookmark(db *sql.DB, id, userId string) func() (*sql.Row, error) {
	return func() (*sql.Row, error) {
		stmt, err := db.Prepare(utils.GET_BOOKMARK)

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

func deleteBookmarkTagIds(execer utils.Execer, id string) error {
	stmt, err := execer.Prepare(utils.DELETE_BOOKMARK_TAG_IDS)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(id); err != nil {
		return err
	}

	return nil
}
