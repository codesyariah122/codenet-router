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

func TestHelperClientAuthenticatedIPC(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretPath, []byte("ipc-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := NewHelperClient("test.sock", secretPath)
	client.dial = func(context.Context, string, string) (net.Conn, error) {
		c, s := net.Pipe()
		go func() {
			defer s.Close()
			var request helperRequest
			if err := json.NewDecoder(bufio.NewReader(s)).Decode(&request); err != nil {
				return
			}
			if request.Operation != "list" || request.MAC != signRequest([]byte("ipc-secret"), request) {
				return
			}
			_ = json.NewEncoder(s).Encode(helperResponse{Success: true, Data: json.RawMessage(`[]`)})
		}()
		return c, nil
	}
	users, err := client.List(context.Background())
	if err != nil || len(users) != 0 {
		t.Fatalf("users=%#v err=%v", users, err)
	}
}

func TestReplayCacheRejectsReuseAndExpires(t *testing.T) {
	cache := newReplayCache()
	now := time.Now()
	if !cache.Accept("00112233445566778899aabbccddeeff", now) {
		t.Fatal("nonce pertama ditolak")
	}
	if cache.Accept("00112233445566778899aabbccddeeff", now) {
		t.Fatal("nonce replay diterima")
	}
	if !cache.Accept("11112222333344445555666677778888", now.Add(helperMaxAge+time.Second)) {
		t.Fatal("nonce setelah expiry ditolak")
	}
}

func TestSecretAndSocketPathSecurity(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readSecret(secret); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("secret permission lemah diterima: %v", err)
	}
	if err := os.Chmod(secret, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSecret(secret); err != nil {
		t.Fatalf("secret owner-only ditolak: %v", err)
	}
	link := filepath.Join(dir, "link.sock")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatal(err)
	}
	if err := prepareSocketPath(link); err == nil {
		t.Fatal("symlink socket diterima")
	}
}

func TestServeConnRejectsInvalidReplayExpiredAndOversizedRequests(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "chap-secrets"))
	secret := []byte("secret")
	replay := newReplayCache()
	call := func(reqBytes []byte) helperResponse {
		client, server := net.Pipe()
		done := make(chan struct{})
		go func() { serveConn(server, secret, store, replay); close(done) }()
		writeDone := make(chan struct{})
		go func() { _, _ = client.Write(reqBytes); close(writeDone) }()
		var response helperResponse
		_ = json.NewDecoder(bufio.NewReader(client)).Decode(&response)
		_ = client.Close()
		<-writeDone
		<-done
		return response
	}
	valid := helperRequest{Operation: "list", Nonce: "00112233445566778899aabbccddeeff", Timestamp: time.Now().Unix()}
	valid.MAC = signRequest(secret, valid)
	encoded, _ := json.Marshal(valid)
	encoded = append(encoded, '\n')
	if response := call(encoded); !response.Success {
		t.Fatal("request valid ditolak")
	}
	if response := call(encoded); response.Success {
		t.Fatal("replay diterima")
	}
	expired := helperRequest{Operation: "list", Nonce: "11112222333344445555666677778888", Timestamp: time.Now().Add(-time.Hour).Unix()}
	expired.MAC = signRequest(secret, expired)
	expiredBytes, _ := json.Marshal(expired)
	expiredBytes = append(expiredBytes, '\n')
	if response := call(expiredBytes); response.Success {
		t.Fatal("request expired diterima")
	}
	if response := call([]byte(strings.Repeat("x", helperRequestLimit+1))); response.Success {
		t.Fatal("request oversized diterima")
	}
}
