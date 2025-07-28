package db

import (
	"database/sql"
	"log"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
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

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

	CREATE TRIGGER IF NOT EXISTS update_users_updated_at
		AFTER UPDATE ON users
		FOR EACH ROW
		WHEN NEW.updated_at = OLD.updated_at
	BEGIN
		UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;
	`

	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	log.Println("Database initialized successfully")
	return db, nil
}
