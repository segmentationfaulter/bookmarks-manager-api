package tags

import (
	"strings"
	"time"

	"github.com/segmentationfaulter/bookmarks-manager-api/internal/utils"
)

type Tag struct {
	Id        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
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

func GetTags(execer utils.Execer, tags []string, userId string) ([]Tag, error) {
	if len(tags) == 0 {
		return []Tag{}, nil
	}

	query := "SELECT id, name, created_at FROM tags WHERE user_id = ? AND name IN ("
	placeholders := make([]string, len(tags))
	for i := range tags {
		placeholders[i] = "?"
	}
	query = query + strings.Join(placeholders, ", ") + ")"

	args := []any{userId}
	for _, tag := range tags {
		args = append(args, tag)
	}
	stmt, err := execer.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
