package tags

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/segmentationfaulter/bookmarks-manager-api/internal/utils"
)

type Tag struct {
	Id        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func GetTagsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, httpStatus, err := utils.IsAuthenticated(r)
		if err != nil {
			http.Error(w, err.Error(), httpStatus)
			return
		}

		tags, err := GetTags(db, nil, string(userId))

		if err != nil {
			http.Error(w, "Error getting tags: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tags)
	}
}

func CreateTags(tx utils.Execer, tags []string, userId string) error {
	if len(tags) == 0 {
		return nil
	}

	query := "INSERT OR IGNORE INTO tags (user_id, name) VALUES"
	placeholders := make([]string, len(tags))
	for i := range tags {
		placeholders[i] = "(?, ?)"
	}
	query = query + " " + strings.Join(placeholders, ", ") + ";"

	var args []any
	for _, tag := range tags {
		args = append(args, userId, tag)
	}

	_, err := utils.Exec(tx, query, args...)

	return err
}

func tagsQueryRunner(execer utils.Execer, tags []string, userId string) func() (*sql.Stmt, *sql.Rows, error) {
	return func() (*sql.Stmt, *sql.Rows, error) {
		baseQuery := "SELECT id, name, created_at FROM tags WHERE user_id = ?"
		var tagsFilter string

		if len(tags) > 0 {
			tagsFilter = " AND name IN ("
			placeholders := make([]string, len(tags))
			for i := range tags {
				placeholders[i] = "?"
			}

			tagsFilter = tagsFilter + strings.Join(placeholders, ", ") + ")"
		}

		query := baseQuery + tagsFilter

		args := []any{userId}
		for _, tag := range tags {
			args = append(args, tag)
		}
		stmt, err := execer.Prepare(query)
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

func tagsScanner(rows *sql.Rows) ([]Tag, error) {
	var tagsResult []Tag

	for rows.Next() {
		tag := Tag{}
		if err := rows.Scan(&tag.Id, &tag.Name, &tag.CreatedAt); err != nil {
			return nil, err
		}
		tagsResult = append(tagsResult, tag)
	}

	return tagsResult, rows.Err()
}

func GetTags(execer utils.Execer, tags []string, userId string) ([]Tag, error) {
	return utils.FindMany(tagsQueryRunner(execer, tags, userId), tagsScanner)
}

func UpdateBookmarkTags(execer utils.Execer, bookmarkId int64, tagIds []int) error {
	if len(tagIds) == 0 {
		return nil
	}
	query := "INSERT OR IGNORE INTO bookmark_tags (bookmark_id, tag_id) VALUES "
	placeholders := make([]string, len(tagIds))
	for i := range tagIds {
		placeholders[i] = "(?, ?)"
	}

	query = query + strings.Join(placeholders, ",")

	var args []any
	for _, tagId := range tagIds {
		args = append(args, bookmarkId, tagId)
	}

	_, err := utils.Exec(execer, query, args...)
	return err
}

func TagIds(tags []Tag) []int {
	result := make([]int, len(tags))

	for i, tag := range tags {
		result[i] = tag.Id
	}

	return result
}
