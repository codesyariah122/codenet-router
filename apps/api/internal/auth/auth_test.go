package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

const secureTestPassword = "Correct-Horse-Battery-99!"

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := OpenStore(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestPasswordArgon2id(t *testing.T) {
	hash, err := HashPassword(secureTestPassword)
	if err != nil {
		t.Fatal(err)
	}
	if hash == secureTestPassword || !VerifyPassword(hash, secureTestPassword) || VerifyPassword(hash, "salah") {
		t.Fatal("verifikasi Argon2id tidak sesuai")
	}
}

func TestResetPasswordMembatalkanSession(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.CreateAdmin(ctx, "admin", secureTestPassword); err != nil {
		t.Fatal(err)
	}
	service := NewService(store, Config{})
	result, err := service.Login(ctx, "admin", secureTestPassword, "127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdatePassword(ctx, "admin", "Different-Secure-Password-88!"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(ctx, result.RawToken); err != ErrSessionInvalid {
		t.Fatalf("session lama belum dibatalkan: %v", err)
	}
}

func TestSessionExpirationStore(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.CreateAdmin(ctx, "admin", secureTestPassword); err != nil {
		t.Fatal(err)
	}
	service := NewService(store, Config{SessionTTL: time.Hour})
	result, err := service.Login(ctx, "admin", secureTestPassword, "127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return result.Session.ExpiresAt.Add(time.Second) }
	if _, err := service.Authenticate(ctx, result.RawToken); err != ErrSessionInvalid {
		t.Fatalf("session expired masih valid: %v", err)
	}
}

func TestAdminHanyaDapatDiinisialisasiSekali(t *testing.T) {
	store := newTestStore(t)
	if err := store.CreateAdmin(context.Background(), "admin", secureTestPassword); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAdmin(context.Background(), "admin2", "Another-Secure-Password-77!"); err != ErrAdminExists {
		t.Fatalf("mengharapkan ErrAdminExists, mendapat %v", err)
	}
}
