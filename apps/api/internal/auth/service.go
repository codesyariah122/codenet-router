package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"strings"
	"sync"
	"time"
)

var ErrRateLimited = errors.New("terlalu banyak percobaan login; coba kembali nanti")

type Config struct {
	SessionTTL       time.Duration
	LoginMaxAttempts int
	LoginWindow      time.Duration
}

type LoginResult struct {
	Session  Session
	RawToken string
}

type attempt struct {
	Count int
	Reset time.Time
}

type Service struct {
	store    *Store
	config   Config
	now      func() time.Time
	mu       sync.Mutex
	attempts map[string]attempt
}

func NewService(store *Store, config Config) *Service {
	if config.SessionTTL <= 0 {
		config.SessionTTL = 8 * time.Hour
	}
	if config.LoginMaxAttempts <= 0 {
		config.LoginMaxAttempts = 5
	}
	if config.LoginWindow <= 0 {
		config.LoginWindow = 15 * time.Minute
	}
	return &Service{store: store, config: config, now: time.Now, attempts: map[string]attempt{}}
}

func (s *Service) Login(ctx context.Context, username, password, remoteAddress string) (LoginResult, error) {
	keys := loginKeys(username, remoteAddress)
	if !s.allowAll(keys) {
		return LoginResult{}, ErrRateLimited
	}
	admin, err := s.store.VerifyAdmin(ctx, username, password)
	if err != nil {
		s.recordFailure(keys)
		return LoginResult{}, ErrInvalidCredentials
	}
	s.clearFailures(keys)
	rawToken, tokenHash, err := generateSecret()
	if err != nil {
		return LoginResult{}, err
	}
	csrfToken, _, err := generateSecret()
	if err != nil {
		return LoginResult{}, err
	}
	expiresAt := s.now().Add(s.config.SessionTTL)
	if err := s.store.CreateSession(ctx, admin.ID, tokenHash, csrfToken, expiresAt); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Session: Session{Admin: admin, CSRFToken: csrfToken, ExpiresAt: expiresAt}, RawToken: rawToken}, nil
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (Session, error) {
	if rawToken == "" {
		return Session{}, ErrSessionInvalid
	}
	hash := sha256.Sum256([]byte(rawToken))
	return s.store.SessionByTokenHash(ctx, hash[:])
}

func (s *Service) Logout(ctx context.Context, sessionID int64) error {
	return s.store.DeleteSession(ctx, sessionID)
}

func generateSecret() (string, []byte, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", nil, err
	}
	raw := base64.RawURLEncoding.EncodeToString(buffer)
	hash := sha256.Sum256([]byte(raw))
	return raw, hash[:], nil
}

func loginKeys(username, remoteAddress string) []string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		host = remoteAddress
	}
	return []string{"user|" + strings.ToLower(strings.TrimSpace(username)), "ip|" + host}
}

func (s *Service) allowAll(keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for _, key := range keys {
		item, exists := s.attempts[key]
		if exists && now.After(item.Reset) {
			delete(s.attempts, key)
			continue
		}
		if exists && item.Count >= s.config.LoginMaxAttempts {
			return false
		}
	}
	return true
}

func (s *Service) recordFailure(keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for _, key := range keys {
		item := s.attempts[key]
		if item.Reset.IsZero() || now.After(item.Reset) {
			item = attempt{Reset: now.Add(s.config.LoginWindow)}
		}
		item.Count++
		s.attempts[key] = item
	}
}

func (s *Service) clearFailures(keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range keys {
		delete(s.attempts, key)
	}
}
