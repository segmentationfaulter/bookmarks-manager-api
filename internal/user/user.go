package user

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/segmentationfaulter/bookmarks-manager-api/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type SearchFlag uint8

const (
	MAX_USERNAME_LENGTH = 50
	MIN_USERNAME_LENGTH = 3
	MIN_PASSWORD_LENGTH = 8
)

const (
	SEARCH_BY_USERNAME SearchFlag = 1 << iota
	SEARCH_BY_ID
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

func ProfileHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, status, err := utils.IsAuthenticated(r)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}
		user, status, err := utils.FindOne(findUser(db, SEARCH_BY_ID, string(userId)), userScanner)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user.public())
	}
}

func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := utils.DecodeRequestBody[User](r)
		if err != nil {
			http.Error(w, "Error decoding request body", http.StatusBadRequest)
			return
		}

		if user.Username == "" || user.Password == "" {
			http.Error(w, "Invalid login credentials", http.StatusBadRequest)
			return
		}

		savedUser, status, err := utils.FindOne(findUser(db, SEARCH_BY_USERNAME, user.Username), userScanner)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(savedUser.Password), []byte(user.Password))
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		token, err := jwt.
			NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"sub": strconv.Itoa(savedUser.Id),
			}).
			SignedString(utils.SigningKey())

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			User  PublicUser `json:"user"`
			Token string     `json:"token"`
		}{User: savedUser.public(), Token: token})
	}
}

func RegisterationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := utils.DecodeRequestBody[User](r)
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
	if _, err := utils.Exec(db, utils.CREATE_USER, u.Username, u.Email, string(hash)); err != nil {
		return err
	}
	return nil
}

func userScanner(row *sql.Row) (*User, error) {
	user := new(User)
	err := row.Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	return user, err
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

func findUser(db *sql.DB, searchFlag SearchFlag, queryValue string) func() (*sql.Row, error) {
	return func() (*sql.Row, error) {
		var stmt *sql.Stmt
		var err error

		if searchFlag&SEARCH_BY_USERNAME != 0 {
			stmt, err = db.Prepare("SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE username= ?")
		} else if searchFlag&SEARCH_BY_ID != 0 {
			stmt, err = db.Prepare("SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE id= ?")
		}

		if err != nil {
			return nil, err
		}
		return stmt.QueryRow(queryValue), nil
	}
}
