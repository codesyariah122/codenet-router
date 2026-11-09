package pppoeusers

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
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
