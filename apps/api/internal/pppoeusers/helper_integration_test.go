package pppoeusers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func startFixtureHelper(t *testing.T) (*HelperClient, string, []byte, context.CancelFunc) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("mode fixture sengaja menolak root")
	}
	root := "/var/lib/codenet-v04/helper-test"
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Skipf("fixture root tidak dapat dibuat: %v", err)
	}
	dir, err := os.MkdirTemp(root, "integration-")
	if err != nil {
		t.Skipf("fixture dir tidak dapat dibuat: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	chap := filepath.Join(dir, "chap-secrets")
	if err := os.WriteFile(chap, []byte("alice * oldsecret pool-main 10000/5000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secretPath := filepath.Join(dir, "helper.secret")
	secret := []byte("integration-secret")
	if err := os.WriteFile(secretPath, secret, 0o600); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "helper.sock")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = ServeWithConfig(ctx, ServeConfig{SocketPath: socket, SecretFile: secretPath, ChapSecrets: chap, Fixture: true, SocketUID: -1, SocketGID: -1, SocketMode: 0o600})
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if info, statErr := os.Stat(socket); statErr == nil && info.Mode()&os.ModeSocket != 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	client := NewHelperClient(socket, secretPath)
	return client, chap, secret, cancel
}

func TestFixtureHelperEndToEndCRUD(t *testing.T) {
	client, chap, _, cancel := startFixtureHelper(t)
	defer cancel()
	ctx := context.Background()
	users, err := client.List(ctx)
	if err != nil || len(users) != 1 {
		t.Fatalf("list=%#v err=%v", users, err)
	}
	created, err := client.Create(ctx, UserInput{Username: "bob", Password: "password123", Server: "*", IPPool: "pool-main", DownloadBPS: 20000000, UploadBPS: 10000000})
	if err != nil || created.Username != "bob" {
		t.Fatalf("create=%#v err=%v", created, err)
	}
	if _, err := client.Update(ctx, "bob", UserInput{Server: "*", IPPool: "pool-updated", DownloadBPS: 30000000, UploadBPS: 15000000}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.SetEnabled(ctx, "bob", false); err != nil {
		t.Fatal(err)
	}
	if _, err := client.SetEnabled(ctx, "bob", true); err != nil {
		t.Fatal(err)
	}
	if err := client.Delete(ctx, "bob"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(chap)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "bob") {
		t.Fatalf("delete tidak tersimpan: %s", data)
	}
}

func TestFixtureHelperRejectsSecurityFailures(t *testing.T) {
	client, _, secret, cancel := startFixtureHelper(t)
	defer cancel()
	conn, err := net.Dial("unix", client.SocketPath)
	if err != nil {
		t.Fatal(err)
	}
	bad := helperRequest{Operation: "list", Nonce: "00112233445566778899aabbccddeeff", Timestamp: time.Now().Unix(), MAC: "00"}
	_ = json.NewEncoder(conn).Encode(bad)
	var response helperResponse
	_ = json.NewDecoder(bufio.NewReader(conn)).Decode(&response)
	_ = conn.Close()
	if response.Success {
		t.Fatal("HMAC invalid diterima")
	}

	request := helperRequest{Operation: "list", Nonce: "11112222333344445555666677778888", Timestamp: time.Now().Unix()}
	request.MAC = signRequest(secret, request)
	encoded, _ := json.Marshal(request)
	encoded = append(encoded, '\n')
	for attempt := 0; attempt < 2; attempt++ {
		conn, err = net.Dial("unix", client.SocketPath)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = conn.Write(encoded)
		response = helperResponse{}
		_ = json.NewDecoder(bufio.NewReader(conn)).Decode(&response)
		_ = conn.Close()
		if attempt == 0 && !response.Success {
			t.Fatal("request pertama ditolak")
		}
		if attempt == 1 && response.Success {
			t.Fatal("replay request diterima")
		}
	}

	conn, err = net.Dial("unix", client.SocketPath)
	if err != nil {
		t.Fatal(err)
	}
	expired := helperRequest{Operation: "list", Nonce: "22223333444455556666777788889999", Timestamp: time.Now().Add(-time.Hour).Unix()}
	expired.MAC = signRequest(secret, expired)
	_ = json.NewEncoder(conn).Encode(expired)
	response = helperResponse{}
	_ = json.NewDecoder(bufio.NewReader(conn)).Decode(&response)
	_ = conn.Close()
	if response.Success {
		t.Fatal("timestamp expired diterima")
	}

	conn, err = net.Dial("unix", client.SocketPath)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = conn.Write([]byte(strings.Repeat("x", helperRequestLimit+1)))
	response = helperResponse{}
	_ = json.NewDecoder(bufio.NewReader(conn)).Decode(&response)
	_ = conn.Close()
	if response.Success {
		t.Fatal("request terlalu besar diterima")
	}
	if _, err := NewHelperClient(filepath.Join(t.TempDir(), "missing.sock"), filepath.Join(t.TempDir(), "missing.secret")).List(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("socket unavailable err=%v", err)
	}
}

func TestFixtureModeRejectsProductionPath(t *testing.T) {
	if err := validateHelperConfig(ServeConfig{SocketPath: filepath.Join(t.TempDir(), "helper.sock"), SecretFile: filepath.Join(t.TempDir(), "secret"), ChapSecrets: "/etc/codenet/chap-secrets", Fixture: true}); err == nil {
		t.Fatal("fixture mengizinkan path produksi")
	}
}
