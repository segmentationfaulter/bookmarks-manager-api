package user

import (
	"database/sql"
	"encoding/json"
	"errors"
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

type PublicUser struct {
	Id        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	PublicUser
	Password string
}

func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := parse(r)
		if err != nil {
			http.Error(w, "Error decoding request body", http.StatusBadRequest)
			return
		}

		if user.Username == "" || user.Password == "" {
			http.Error(w, "Invalid login credentials", http.StatusBadRequest)
			return
		}

		savedUser, status, err := user.find(db)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(savedUser.Password), []byte(user.Password))
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(savedUser.public())
		w.WriteHeader(http.StatusOK)
	}
}

func RegisterationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := parse(r)
		if err != nil {
			http.Error(w, "Error decoding request body", http.StatusBadRequest)
			return
		}

		err = user.validate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = user.save(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func parse(r *http.Request) (User, error) {
	user := User{}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (u *User) validate() error {
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

func (u *User) save(db *sql.DB) error {
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

func (u *User) find(db *sql.DB) (*User, int, error) {
	stmt, err := db.Prepare("SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE username= ?")
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	defer stmt.Close()

	var savedUser = User{}

	err = stmt.QueryRow(u.Username).Scan(
		&savedUser.Id,
		&savedUser.Username,
		&savedUser.Email,
		&savedUser.Password,
		&savedUser.CreatedAt,
		&savedUser.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusUnauthorized, errors.New("Invalid credentials")
		}
		return nil, http.StatusInternalServerError, err
	}

	return &savedUser, http.StatusOK, nil
}

func (u *User) public() PublicUser {
	return PublicUser{
		Id:        u.Id,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
