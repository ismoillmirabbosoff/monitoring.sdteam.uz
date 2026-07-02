package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("topilmadi")

type User struct {
	ID              int       `json:"id"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"-"`
	Name            string    `json:"name"`
	Role            string    `json:"role"`
	Ext             string    `json:"ext"`
	Active          bool      `json:"active"`
	InitialPassword string    `json:"initial_password"` // admin ko'rishi uchun ochiq parol
	CreatedAt       time.Time `json:"created_at"`
}

// CreateUser yangi foydalanuvchi yaratadi. plainPassword — admin ko'rishi uchun ochiq
// saqlanadi (initial_password); bo'sh bo'lsa saqlanmaydi.
func (s *Store) CreateUser(ctx context.Context, email, hash, name, role, ext, plainPassword string) (User, error) {
	var u User
	var extVal *string
	if ext != "" {
		extVal = &ext
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name, role, ext, initial_password)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING `+userCols,
		email, hash, name, role, extVal, plainPassword).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.Ext, &u.Active, &u.InitialPassword, &u.CreatedAt)
	return u, err
}

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.Ext, &u.Active, &u.InitialPassword, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

const userCols = `id, email, password_hash, name, role, COALESCE(ext,''), active, COALESCE(initial_password,''), created_at`

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE lower(email) = lower($1)`, email))
}

// UserByExt operator extension bo'yicha foydalanuvchini qaytaradi (ext+parol login uchun).
func (s *Store) UserByExt(ctx context.Context, ext string) (User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE ext = $1 LIMIT 1`, ext))
}

func (s *Store) UserByID(ctx context.Context, id int) (User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id))
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+userCols+` FROM users ORDER BY role DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&n)
	return n, err
}

// UpdateUser nil bo'lmagan maydonlarni yangilaydi.
// plainPassword nil bo'lmasa initial_password ham yangilanadi (admin ko'rishi uchun).
func (s *Store) UpdateUser(ctx context.Context, id int, name, role, ext *string, active *bool, passwordHash, plainPassword *string) (User, error) {
	if name != nil {
		s.pool.Exec(ctx, `UPDATE users SET name=$1 WHERE id=$2`, *name, id)
	}
	if role != nil {
		s.pool.Exec(ctx, `UPDATE users SET role=$1 WHERE id=$2`, *role, id)
	}
	if ext != nil {
		var e *string
		if *ext != "" {
			e = ext
		}
		s.pool.Exec(ctx, `UPDATE users SET ext=$1 WHERE id=$2`, e, id)
	}
	if active != nil {
		s.pool.Exec(ctx, `UPDATE users SET active=$1 WHERE id=$2`, *active, id)
	}
	if passwordHash != nil {
		s.pool.Exec(ctx, `UPDATE users SET password_hash=$1 WHERE id=$2`, *passwordHash, id)
	}
	if plainPassword != nil {
		s.pool.Exec(ctx, `UPDATE users SET initial_password=$1 WHERE id=$2`, *plainPassword, id)
	}
	return s.UserByID(ctx, id)
}

func (s *Store) DeleteUser(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// ---- Login kodlari ----

func (s *Store) CreateLoginCode(ctx context.Context, userID int, code string, ttl time.Duration) error {
	// eski kodlarni tozalaymiz
	s.pool.Exec(ctx, `DELETE FROM login_codes WHERE user_id = $1`, userID)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO login_codes (user_id, code, expires_at) VALUES ($1,$2,$3)`,
		userID, code, time.Now().Add(ttl))
	return err
}

// CheckLoginCode kod to'g'ri va amal qilsa true qaytaradi va kodni o'chiradi.
func (s *Store) CheckLoginCode(ctx context.Context, userID int, code string) (bool, error) {
	var id int
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM login_codes WHERE user_id=$1 AND code=$2 AND expires_at > now() ORDER BY id DESC LIMIT 1`,
		userID, code).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	s.pool.Exec(ctx, `DELETE FROM login_codes WHERE user_id = $1`, userID)
	return true, nil
}

// ---- Sessiyalar ----

type Session struct {
	Token     string    `json:"token"`
	UserID    int       `json:"user_id"`
	UserAgent string    `json:"user_agent"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	// qo'shimcha (ro'yxat uchun)
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	UserRole  string `json:"user_role"`
}

func (s *Store) CreateSession(ctx context.Context, token string, userID int, ua, ip string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (token, user_id, user_agent, ip) VALUES ($1,$2,$3,$4)`,
		token, userID, ua, ip)
	return err
}

// SessionUser sessiya token'i bo'yicha foydalanuvchini qaytaradi (last_seen yangilanadi).
func (s *Store) SessionUser(ctx context.Context, token string) (User, error) {
	var uid int
	err := s.pool.QueryRow(ctx, `SELECT user_id FROM sessions WHERE token = $1`, token).Scan(&uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	s.pool.Exec(ctx, `UPDATE sessions SET last_seen = now() WHERE token = $1`, token)
	return s.UserByID(ctx, uid)
}

func (s *Store) ListSessions(ctx context.Context) ([]Session, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.token, s.user_id, s.user_agent, s.ip, s.created_at, s.last_seen,
		       u.name, u.email, u.role
		FROM sessions s JOIN users u ON u.id = s.user_id
		ORDER BY s.last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.Token, &s.UserID, &s.UserAgent, &s.IP, &s.CreatedAt, &s.LastSeen,
			&s.UserName, &s.UserEmail, &s.UserRole); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}
