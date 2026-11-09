package pppoeusers

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
)

type helperRequest struct {
	Operation string     `json:"operation"`
	Username  string     `json:"username,omitempty"`
	Input     *UserInput `json:"input,omitempty"`
	Enabled   *bool      `json:"enabled,omitempty"`
	Nonce     string     `json:"nonce"`
	MAC       string     `json:"mac"`
}
type helperResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type HelperClient struct {
	SocketPath string
	SecretFile string
	Timeout    time.Duration
	dial       func(context.Context, string, string) (net.Conn, error)
}

func NewHelperClient(socketPath, secretFile string) *HelperClient {
	return &HelperClient{SocketPath: socketPath, SecretFile: secretFile, Timeout: 3 * time.Second, dial: (&net.Dialer{}).DialContext}
}

func (c *HelperClient) call(ctx context.Context, req helperRequest, target any) error {
	if c == nil || c.SocketPath == "" || c.SecretFile == "" {
		return ErrUnavailable
	}
	secret, err := readSecret(c.SecretFile)
	if err != nil {
		return ErrUnavailable
	}
	nonceBytes := make([]byte, 16)
	copy(nonceBytes, []byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	req.Nonce = hex.EncodeToString(nonceBytes)
	req.MAC = signRequest(secret, req)
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dial := c.dial
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	conn, err := dial(callCtx, "unix", c.SocketPath)
	if err != nil {
		return ErrUnavailable
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return ErrUnavailable
	}
	var response helperResponse
	limited := &ioLimitConn{Conn: conn, Remaining: 1 << 20}
	if err := json.NewDecoder(bufio.NewReader(limited)).Decode(&response); err != nil {
		return ErrUnavailable
	}
	if !response.Success {
		if response.Message == "" {
			return ErrUnavailable
		}
		return errors.New(response.Message)
	}
	if target != nil && len(response.Data) > 0 {
		return json.Unmarshal(response.Data, target)
	}
	return nil
}

func (c *HelperClient) List(ctx context.Context) ([]models.PPPoEUser, error) {
	var out []models.PPPoEUser
	err := c.call(ctx, helperRequest{Operation: "list"}, &out)
	return out, err
}
func (c *HelperClient) Create(ctx context.Context, input UserInput) (models.PPPoEUser, error) {
	var out models.PPPoEUser
	err := c.call(ctx, helperRequest{Operation: "create", Input: &input}, &out)
	return out, err
}
func (c *HelperClient) Update(ctx context.Context, username string, input UserInput) (models.PPPoEUser, error) {
	var out models.PPPoEUser
	err := c.call(ctx, helperRequest{Operation: "update", Username: username, Input: &input}, &out)
	return out, err
}
func (c *HelperClient) SetEnabled(ctx context.Context, username string, enabled bool) (models.PPPoEUser, error) {
	var out models.PPPoEUser
	err := c.call(ctx, helperRequest{Operation: "set_enabled", Username: username, Enabled: &enabled}, &out)
	return out, err
}
func (c *HelperClient) Delete(ctx context.Context, username string) error {
	return c.call(ctx, helperRequest{Operation: "delete", Username: username}, nil)
}

func readSecret(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, ErrUnavailable
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 || len(data) > 4096 {
		return nil, ErrUnavailable
	}
	return []byte(strings.TrimSpace(string(data))), nil
}
func signRequest(secret []byte, req helperRequest) string {
	req.MAC = ""
	data, _ := json.Marshal(req)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

type ioLimitConn struct {
	net.Conn
	Remaining int
}

func (r *ioLimitConn) Read(p []byte) (int, error) {
	if r.Remaining <= 0 {
		return 0, errors.New("respons helper terlalu besar")
	}
	if len(p) > r.Remaining {
		p = p[:r.Remaining]
	}
	n, err := r.Conn.Read(p)
	r.Remaining -= n
	return n, err
}

// Serve menjalankan helper terpisah. Helper tidak diakses dari HTTP dan hanya
// menerima lima operasi terstruktur yang divalidasi oleh FileStore.
func Serve(ctx context.Context, socketPath, secretFile, chapSecrets string) error {
	if filepath.Clean(chapSecrets) != "/etc/codenet/chap-secrets" {
		return errors.New("helper hanya mengizinkan /etc/codenet/chap-secrets")
	}
	secret, err := readSecret(secretFile)
	if err != nil {
		return err
	}
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer listener.Close()
	_ = os.Chmod(socketPath, 0o660)
	store := NewFileStore(chapSecrets)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}
		go serveConn(conn, secret, store)
	}
}

func serveConn(conn net.Conn, secret []byte, store *FileStore) {
	defer conn.Close()
	var req helperRequest
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&req); err != nil || !hmac.Equal([]byte(req.MAC), []byte(signRequest(secret, req))) {
		_ = json.NewEncoder(conn).Encode(helperResponse{Message: "permintaan helper tidak valid"})
		return
	}
	var data any
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	switch req.Operation {
	case "list":
		data, err = store.List(ctx)
	case "create":
		if req.Input == nil {
			err = ErrInvalid
		} else {
			data, err = store.Create(ctx, *req.Input)
		}
	case "update":
		if req.Input == nil {
			err = ErrInvalid
		} else {
			data, err = store.Update(ctx, req.Username, *req.Input)
		}
	case "set_enabled":
		if req.Enabled == nil {
			err = ErrInvalid
		} else {
			data, err = store.SetEnabled(ctx, req.Username, *req.Enabled)
		}
	case "delete":
		err = store.Delete(ctx, req.Username)
	default:
		err = ErrInvalid
	}
	response := helperResponse{Success: err == nil}
	if err != nil {
		response.Message = err.Error()
	} else if data != nil {
		response.Data, _ = json.Marshal(data)
	}
	_ = json.NewEncoder(conn).Encode(response)
}
