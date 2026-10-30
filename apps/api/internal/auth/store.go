package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrAdminExists        = errors.New("akun administrator sudah diinisialisasi")
	ErrAdminNotFound      = errors.New("administrator tidak ditemukan")
	ErrInvalidCredentials = errors.New("username atau password tidak valid")
	ErrSessionInvalid     = errors.New("session tidak valid atau sudah berakhir")
	usernamePattern       = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,64}$`)
)

type Admin struct {
	ID       int64
	Username string
}

type Session struct {
	ID        int64
	Admin     Admin
	CSRFToken string
	ExpiresAt time.Time
}

type Store struct {
	db        *sql.DB
	now       func() time.Time
	dummyHash string
}

func OpenStore(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("path database tidak boleh kosong")
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("membuat direktori database: %w", err)
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("membuka database: %w", err)
	}
	db.SetMaxOpenConns(1)
	dummyHash, err := HashPassword("CODENET timing equalization value 97!")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("menyiapkan verifikasi password: %w", err)
	}
	store := &Store{db: db, now: time.Now, dummyHash: dummyHash}
	if err := store.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		db.Close()
		return nil, fmt.Errorf("mengatur permission database: %w", err)
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA secure_delete = ON`,
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE COLLATE NOCASE,
			password_hash TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			admin_id INTEGER NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
			token_hash BLOB NOT NULL UNIQUE,
			csrf_token TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrasi database autentikasi: %w", err)
		}
	}
	return nil
}

func ValidateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return errors.New("username harus 3-64 karakter dan hanya boleh berisi huruf, angka, titik, garis bawah, atau tanda hubung")
	}
	return nil
}

func (s *Store) AdminCount(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		return 0, fmt.Errorf("menghitung administrator: %w", err)
	}
	return count, nil
}

func (s *Store) CreateAdmin(ctx context.Context, username, password string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if err := ValidateNewPassword(username, password); err != nil {
		return err
	}
	count, err := s.AdminCount(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrAdminExists
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	now := s.now().Unix()
	if _, err := s.db.ExecContext(ctx, `INSERT INTO admins(username, password_hash, created_at, updated_at) VALUES(?, ?, ?, ?)`, username, hash, now, now); err != nil {
		return fmt.Errorf("membuat administrator: %w", err)
	}
	return nil
}

func (s *Store) UpdatePassword(ctx context.Context, username, password string) error {
	if err := ValidateNewPassword(username, password); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE admins SET password_hash = ?, updated_at = ? WHERE username = ?`, hash, s.now().Unix(), username)
	if err != nil {
		return fmt.Errorf("memperbarui password: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrAdminNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE admin_id = (SELECT id FROM admins WHERE username = ?)`, username); err != nil {
		return fmt.Errorf("membatalkan session lama: %w", err)
	}
	return tx.Commit()
}

func (s *Store) VerifyAdmin(ctx context.Context, username, password string) (Admin, error) {
	var admin Admin
	var passwordHash string
	err := s.db.QueryRowContext(ctx, `SELECT id, username, password_hash FROM admins WHERE username = ?`, username).Scan(&admin.ID, &admin.Username, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		VerifyPassword(s.dummyHash, password)
		return Admin{}, ErrInvalidCredentials
	}
	if err != nil || !VerifyPassword(passwordHash, password) {
		return Admin{}, ErrInvalidCredentials
	}
	return admin, nil
}

func (s *Store) CreateSession(ctx context.Context, adminID int64, tokenHash []byte, csrfToken string, expiresAt time.Time) error {
	now := s.now().Unix()
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, now); err != nil {
		return fmt.Errorf("membersihkan session: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO sessions(admin_id, token_hash, csrf_token, created_at, expires_at) VALUES(?, ?, ?, ?, ?)`, adminID, tokenHash, csrfToken, now, expiresAt.Unix()); err != nil {
		return fmt.Errorf("membuat session: %w", err)
	}
	return nil
}

func (s *Store) SessionByTokenHash(ctx context.Context, tokenHash []byte) (Session, error) {
	var session Session
	var expiresAt int64
	err := s.db.QueryRowContext(ctx, `SELECT s.id, a.id, a.username, s.csrf_token, s.expires_at FROM sessions s JOIN admins a ON a.id = s.admin_id WHERE s.token_hash = ? AND s.expires_at > ?`, tokenHash, s.now().Unix()).Scan(&session.ID, &session.Admin.ID, &session.Admin.Username, &session.CSRFToken, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionInvalid
	}
	if err != nil {
		return Session{}, fmt.Errorf("membaca session: %w", err)
	}
	session.ExpiresAt = time.Unix(expiresAt, 0)
	return session, nil
}

func (s *Store) DeleteSession(ctx context.Context, sessionID int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID); err != nil {
		return fmt.Errorf("menghapus session: %w", err)
	}
	return nil
}
