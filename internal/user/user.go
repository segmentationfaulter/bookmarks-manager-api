package user

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ParseUser(w http.ResponseWriter, r *http.Request) (User, error) {
	user := User{}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

// TODO: add validation
func (u User) Validate() {}

func (u User) Save(db *sql.DB) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	stmt, err := db.Prepare("INSERT INTO users (username, email, password_hash) VALUES(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(u.Username, u.Email, hash)
	if err != nil {
		return err
	}
	return nil
}
