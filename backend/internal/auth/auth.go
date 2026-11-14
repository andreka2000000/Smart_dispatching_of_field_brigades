// Package auth owns application users and revocable, opaque database sessions.
// Organizer evaluation runs on a separate internal listener without identity.
package auth

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"magistral/backend/internal/storage/postgres"
	"regexp"
	"strings"
	"time"
)

const CookieName = "mt_session"
const Dispatcher = "DISPATCHER"
const Technician = "TECHNICIAN"
const Subscriber = "SUBSCRIBER"
const Lifetime = 8 * time.Hour

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string { return e.Message }
func denied() *Error {
	return &Error{401, "UNAUTHENTICATED", "Войдите в свой аккаунт."}
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	StaffID  string `json:"staff_id,omitempty"`
	Disabled bool   `json:"disabled"`
}
type NewUser struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
	StaffID  string `json:"staff_id"`
}
type Manager struct{ Store *postgres.Store }

func Random() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, e := rand.Read(salt); e != nil {
		return "", e
	}
	key, e := pbkdf2.Key(sha256.New, password, salt, 600000, 32)
	if e != nil {
		return "", e
	}
	return "pbkdf2-sha256$600000$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}
func Verify(encoded, password string) bool {
	pieces := strings.Split(encoded, "$")
	if len(pieces) != 4 || pieces[0] != "pbkdf2-sha256" || pieces[1] != "600000" {
		return false
	}
	salt, e := base64.RawStdEncoding.DecodeString(pieces[2])
	if e != nil || len(salt) != 16 {
		return false
	}
	expected, e := base64.RawStdEncoding.DecodeString(pieces[3])
	if e != nil || len(expected) != 32 {
		return false
	}
	key, e := pbkdf2.Key(sha256.New, password, salt, 600000, 32)
	return e == nil && subtle.ConstantTimeCompare(key, expected) == 1
}

var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)

func (m *Manager) Create(ctx context.Context, in NewUser) (User, error) {
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.Name = strings.TrimSpace(in.Name)
	if !usernamePattern.MatchString(in.Username) || len([]rune(in.Name)) < 2 || len([]rune(in.Name)) > 100 || len(in.Password) < 12 || len(in.Password) > 256 {
		return User{}, &Error{400, "INVALID_ACCOUNT", "Логин: 3–64 латинских символа. Имя: 2–100 символов. Пароль: 12–256 байт."}
	}
	if in.Role != Dispatcher && in.Role != Technician && in.Role != Subscriber {
		return User{}, &Error{400, "INVALID_ROLE", "Неизвестная роль."}
	}
	if in.Role != Technician {
		in.StaffID = ""
	}
	encoded, e := Hash(in.Password)
	if e != nil {
		return User{}, e
	}
	user := User{ID: "USR-" + Random(), Username: in.Username, Name: in.Name, Role: in.Role, StaffID: in.StaffID}
	e = m.Store.Update(ctx, func(tx pgx.Tx) error {
		if in.Role == Technician {
			var exists bool
			if e := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM staff WHERE id=$1)", in.StaffID).Scan(&exists); e != nil {
				return e
			}
			if !exists {
				return &Error{400, "STAFF_NOT_FOUND", "Выберите сотрудника из справочника."}
			}
		}
		_, e := tx.Exec(ctx, "INSERT INTO app_users(id,username,name,role,staff_id,password_hash) VALUES($1,$2,$3,$4,NULLIF($5,''),$6)", user.ID, user.Username, user.Name, user.Role, user.StaffID, encoded)
		return e
	})
	var conflict *pgconn.PgError
	if errors.As(e, &conflict) && conflict.Code == "23505" {
		return User{}, &Error{409, "ACCOUNT_EXISTS", "Логин или сотрудник уже связан с другим аккаунтом."}
	}
	return user, e
}
func (m *Manager) List(ctx context.Context) ([]User, error) {
	rows, e := m.Store.Pool.Query(ctx, "SELECT id,username,name,role,coalesce(staff_id,''),disabled FROM app_users ORDER BY role,name,id")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if e = rows.Scan(&u.ID, &u.Username, &u.Name, &u.Role, &u.StaffID, &u.Disabled); e != nil {
			return nil, e
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
func (m *Manager) Disable(ctx context.Context, actorID, targetID string, disabled bool) error {
	if actorID == targetID {
		return &Error{409, "OWN_ACCOUNT", "Нельзя изменить доступ своему аккаунту."}
	}
	e := m.Store.Update(ctx, func(tx pgx.Tx) error {
		result, e := tx.Exec(ctx, "UPDATE app_users SET disabled=$2 WHERE id=$1", targetID, disabled)
		if e != nil {
			return e
		}
		if result.RowsAffected() == 0 {
			return &Error{404, "ACCOUNT_NOT_FOUND", "Аккаунт не найден."}
		}
		if disabled {
			_, e = tx.Exec(ctx, "DELETE FROM app_sessions WHERE user_id=$1", targetID)
		}
		return e
	})
	var conflict *pgconn.PgError
	if errors.As(e, &conflict) && conflict.Code == "23505" {
		return &Error{409, "STAFF_LINK_IN_USE", "Сотрудник уже связан с другим активным аккаунтом."}
	}
	return e
}
func (m *Manager) Current(ctx context.Context, token string) (User, error) {
	if len(token) != 43 {
		return User{}, denied()
	}
	digest := sha256.Sum256([]byte(token))
	var u User
	e := m.Store.Pool.QueryRow(ctx, "SELECT u.id,u.username,u.name,u.role,coalesce(u.staff_id,''),u.disabled FROM app_sessions s JOIN app_users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now() AND NOT u.disabled", digest[:]).Scan(&u.ID, &u.Username, &u.Name, &u.Role, &u.StaffID, &u.Disabled)
	if errors.Is(e, pgx.ErrNoRows) {
		return u, denied()
	}
	return u, e
}
func (m *Manager) Logout(ctx context.Context, token string) error {
	digest := sha256.Sum256([]byte(token))
	_, e := m.Store.Pool.Exec(ctx, "DELETE FROM app_sessions WHERE token_hash=$1", digest[:])
	return e
}
func (m *Manager) Login(ctx context.Context, username, password, ip string) (User, string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if len(username) > 64 || len(password) > 256 {
		return User{}, "", &Error{401, "INVALID_CREDENTIALS", "Неверный логин или пароль."}
	}
	// Persistent per-account and per-peer windows also cover unknown usernames.
	for key, limit := range map[string]int{"account:" + username: 10, "peer:" + ip: 100} {
		digest := sha256.Sum256([]byte(key))
		var count int
		e := m.Store.Pool.QueryRow(ctx, `INSERT INTO app_login_limits(key,attempts,reset_at) VALUES($1,1,now()+interval '5 minutes') ON CONFLICT(key) DO UPDATE SET attempts=CASE WHEN app_login_limits.reset_at<=now() THEN 1 ELSE app_login_limits.attempts+1 END,reset_at=CASE WHEN app_login_limits.reset_at<=now() THEN now()+interval '5 minutes' ELSE app_login_limits.reset_at END RETURNING attempts`, fmt.Sprintf("%x", digest)).Scan(&count)
		if e != nil {
			return User{}, "", e
		}
		if count > limit {
			return User{}, "", &Error{429, "LOGIN_RATE_LIMIT", "Слишком много попыток. Повторите вход через пять минут."}
		}
	}
	var u User
	var encoded string
	e := m.Store.Pool.QueryRow(ctx, "SELECT id,username,name,role,coalesce(staff_id,''),disabled,password_hash FROM app_users WHERE username=$1", username).Scan(&u.ID, &u.Username, &u.Name, &u.Role, &u.StaffID, &u.Disabled, &encoded)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return u, "", e
	}
	if encoded == "" {
		encoded = "pbkdf2-sha256$600000$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	}
	valid := Verify(encoded, password)
	if e != nil || !valid || u.Disabled {
		return User{}, "", &Error{401, "INVALID_CREDENTIALS", "Неверный логин или пароль."}
	}
	token := Random()
	digest := sha256.Sum256([]byte(token))
	e = m.Store.Update(ctx, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, "DELETE FROM app_sessions WHERE expires_at<=now()"); e != nil {
			return e
		}
		result, e := tx.Exec(ctx, "INSERT INTO app_sessions(token_hash,user_id,expires_at) SELECT $1,id,now()+interval '8 hours' FROM app_users WHERE id=$2 AND NOT disabled", digest[:], u.ID)
		if e != nil {
			return e
		}
		if result.RowsAffected() != 1 {
			return denied()
		}
		key := sha256.Sum256([]byte("account:" + username))
		_, e = tx.Exec(ctx, "DELETE FROM app_login_limits WHERE key=$1", fmt.Sprintf("%x", key))
		return e
	})
	return u, token, e
}
