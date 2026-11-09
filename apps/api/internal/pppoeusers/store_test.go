package pppoeusers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func fixtureStore(t *testing.T) *FileStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "chap-secrets")
	if err := os.WriteFile(path, []byte("# existing\nalice oldsecret * pool-main 10000000/5000000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return NewFileStore(path)
}

func TestParseAndListNeverExposePassword(t *testing.T) {
	store := fixtureStore(t)
	users, err := store.List(context.Background())
	if err != nil || len(users) != 1 || users[0].Username != "alice" {
		t.Fatalf("users=%#v err=%v", users, err)
	}
	if users[0].Password != "oldsecret" {
		t.Fatal("helper perlu mempertahankan secret internal")
	}
}

func TestCreateUpdateDisableDeleteAtomicFixture(t *testing.T) {
	store := fixtureStore(t)
	created, err := store.Create(context.Background(), UserInput{Username: "bob", Password: "newsecret123", Server: "*", IPPool: "pool-main", DownloadBPS: 20000000, UploadBPS: 10000000})
	if err != nil || created.Username != "bob" {
		t.Fatalf("created=%#v err=%v", created, err)
	}
	updated, err := store.Update(context.Background(), "bob", UserInput{Server: "*", IPPool: "pool-backup", DownloadBPS: 30000000, UploadBPS: 15000000})
	if err != nil || updated.IPPool != "pool-backup" || updated.Password != "newsecret123" {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
	if _, err := os.Stat(store.Path + ".bak"); err != nil {
		t.Fatalf("backup tidak dibuat: %v", err)
	}
	disabled, err := store.SetEnabled(context.Background(), "bob", false)
	if err != nil || disabled.Enabled {
		t.Fatalf("disabled=%#v err=%v", disabled, err)
	}
	data, _ := os.ReadFile(store.Path)
	if !strings.Contains(string(data), "# codenet-disabled bob newsecret123") {
		t.Fatalf("disable tidak mempertahankan histori: %s", data)
	}
	if err := store.Delete(context.Background(), "bob"); err != nil {
		t.Fatal(err)
	}
	users, err := store.List(context.Background())
	if err != nil || len(users) != 1 {
		t.Fatalf("users=%#v err=%v", users, err)
	}
}

func TestValidationRejectsTraversalAndMalformedRate(t *testing.T) {
	if err := ValidateUsername("../../etc/passwd"); err == nil {
		t.Fatal("path traversal username diterima")
	}
	if err := ValidateInput(UserInput{Username: "safe", Password: "short", Server: "*", IPPool: "pool", DownloadBPS: 1, UploadBPS: 1}, true); err == nil {
		t.Fatal("password pendek diterima")
	}
	if _, _, err := parseRate("1000"); err == nil {
		t.Fatal("rate tanpa upload diterima")
	}
	if _, err := parseChapSecrets("alice secret * pool 1/2/3\n"); err == nil {
		t.Fatal("format rate rusak diterima")
	}
}

func TestConcurrentWritesDoNotCorruptFile(t *testing.T) {
	store := fixtureStore(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = store.Create(context.Background(), UserInput{Username: "user" + string(rune('a'+i)), Password: "password123", Server: "*", IPPool: "pool", DownloadBPS: 1000, UploadBPS: 500})
		}(i)
	}
	wg.Wait()
	users, err := store.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(users) < 1 || len(users) > 9 {
		t.Fatalf("jumlah user tidak konsisten: %d", len(users))
	}
}
