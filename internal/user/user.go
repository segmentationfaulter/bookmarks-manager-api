package user

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/segmentationfaulter/bookmarks-manager-api/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type QueryRunner func() (*sql.Row, error)
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
		tokenStr, ok := bearerToken(r)
		if !ok {
			http.Error(w, "Missing/malformed token", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return signingKey(), nil
		})
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		userId, err := token.Claims.GetSubject()
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}
		user, status, err := find(findUser(db, SEARCH_BY_ID, userId))
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

		savedUser, status, err := find(findUser(db, SEARCH_BY_USERNAME, user.Username))
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
			SignedString(signingKey())

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

func find(queryRunner QueryRunner) (*User, int, error) {
	row, err := queryRunner()
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	var user = User{}
	err = row.Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusUnauthorized, errors.New("Invalid credentials")
		}
		return nil, http.StatusInternalServerError, err
	}

	return &user, http.StatusOK, nil
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

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, prefix) {
		return "", false
	}
	return strings.TrimPrefix(auth, prefix), true
}

func signingKey() []byte {
	keyStr := os.Getenv("JWT_SIGNING_KEY")
	if keyStr == "" {
		panic("JWT signing key not set")
	}
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		panic("Decoding of JWT signing key failed")
	}
	return key
}

func findUser(db *sql.DB, searchFlag SearchFlag, queryValue string) QueryRunner {
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
