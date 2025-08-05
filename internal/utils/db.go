package utils

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type Execer interface {
	Prepare(query string) (*sql.Stmt, error)
}

const (
	CREATE_BOOKMARK    = `INSERT INTO bookmarks (user_id, url, title, description, notes) VALUES(?, ?, ?, ?, ?)`
	CREATE_USER        = `INSERT INTO users (username, email, password_hash) VALUES(?, ?, ?)`
	GET_BOOKMARK_BY_ID = `SELECT id, url, title, description, notes, created_at, updated_at FROM bookmarks WHERE id = ? AND user_id = ?`
)

func InitDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./bookmarks.db")
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username VARCHAR(50) NOT NULL UNIQUE,
		email VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS bookmarks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		url TEXT NOT NULL UNIQUE CHECK(url <> ''),
		title VARCHAR(500),
		description VARCHAR(2000),
		notes TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	    UNIQUE(user_id, url)
	);

	CREATE TABLE IF NOT EXISTS tags (
	    id INTEGER PRIMARY KEY AUTOINCREMENT,
	    user_id INTEGER NOT NULL,
	    name VARCHAR(50) NOT NULL CHECK(name <> ''),
	    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	    UNIQUE(user_id, name)
	);

	CREATE TABLE IF NOT EXISTS bookmark_tags (
	    bookmark_id INTEGER NOT NULL,
	    tag_id INTEGER NOT NULL,
	    PRIMARY KEY (bookmark_id, tag_id),
	    FOREIGN KEY (bookmark_id) REFERENCES bookmarks(id) ON DELETE CASCADE,
	    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_bookmarks_user_id ON bookmarks(user_id);
	CREATE INDEX IF NOT EXISTS idx_bookmarks_url ON bookmarks(url);
	CREATE INDEX IF NOT EXISTS idx_tags_user_id ON tags(user_id);
	CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);

	CREATE TRIGGER IF NOT EXISTS update_users_updated_at
		AFTER UPDATE ON users
		FOR EACH ROW
		WHEN NEW.updated_at = OLD.updated_at
	BEGIN
		UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;

	CREATE TRIGGER IF NOT EXISTS update_bookmarks_updated_at
	    AFTER UPDATE ON bookmarks
	    FOR EACH ROW
	    WHEN NEW.updated_at = OLD.updated_at
	BEGIN
	    UPDATE bookmarks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;
	`

	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	log.Println("Database initialized successfully")
	return db, nil
}

func Exec(db Execer, query string, args ...any) (sql.Result, error) {
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	result, err := stmt.Exec(args...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func FindOne[T any](queryRunner func() (*sql.Row, error), rowScanner func(*sql.Row) (*T, error)) (*T, int, error) {
	row, err := queryRunner()
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	result, err := rowScanner(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, errors.New(http.StatusText(http.StatusNotFound))
		}
		return nil, http.StatusInternalServerError, err
	}

	return result, http.StatusOK, nil
}

func FindMany[T any](
	queryRunner func() (*sql.Rows, error),
	scanner func(*sql.Rows) []T,
) ([]T, error) {
	rows, err := queryRunner()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := scanner(rows)

	return result, rows.Err()
}
