package user

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	MAX_USERNAME_LENGTH = 50
	MIN_USERNAME_LENGTH = 3
	MIN_PASSWORD_LENGTH = 8
)

type User struct {
	Id        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func RegisterationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := Parse(w, r)
		if err != nil {
			http.Error(w, "Error decoding request body", http.StatusBadRequest)
			return
		}

		err = user.Validate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = user.Save(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func Parse(w http.ResponseWriter, r *http.Request) (User, error) {
	user := User{}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (u *User) Validate() error {
	u.Username = strings.TrimSpace(u.Username)
	if len := utf8.RuneCountInString(u.Username); len < MIN_USERNAME_LENGTH || len > MAX_USERNAME_LENGTH {
		return fmt.Errorf("Username should be no longer than %d characters and less than %d", MAX_USERNAME_LENGTH, MIN_USERNAME_LENGTH)
	}

	u.Email = strings.TrimSpace(u.Email)
	if _, err := mail.ParseAddress(u.Email); err != nil {
		return fmt.Errorf("Invalid email address: %s", u.Email)
	}

	if len := utf8.RuneCountInString(u.Password); len < MIN_PASSWORD_LENGTH {
		return fmt.Errorf("Password should not be less than %d characters", MIN_PASSWORD_LENGTH)
	}

	return nil
}

func (u *User) Save(db *sql.DB) error {
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
