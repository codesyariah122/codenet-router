package pppoeusers

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
)

type helperRequest struct {
	Operation string     `json:"operation"`
	Username  string     `json:"username,omitempty"`
	Input     *UserInput `json:"input,omitempty"`
	Enabled   *bool      `json:"enabled,omitempty"`
	Nonce     string     `json:"nonce"`
	Timestamp int64      `json:"timestamp"`
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

const (
	helperRequestLimit  = 64 * 1024
	helperResponseLimit = 1 << 20
	helperMaxAge        = 30 * time.Second
	helperMaxNonces     = 4096
)

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
	if _, err := rand.Read(nonceBytes); err != nil {
		return ErrUnavailable
	}
	req.Nonce = hex.EncodeToString(nonceBytes)
	req.Timestamp = time.Now().UTC().Unix()
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
	limited := &ioLimitConn{Conn: conn, Remaining: helperResponseLimit}
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
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || !ownedByCurrentUser(info) {
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

func ownedByCurrentUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return os.Geteuid() == 0
	}
	if os.Geteuid() == 0 {
		return stat.Uid == 0
	}
	return uint32(os.Geteuid()) == stat.Uid
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

type ServeConfig struct {
	SocketPath  string
	SecretFile  string
	ChapSecrets string
	Fixture     bool
	SocketUID   int
	SocketGID   int
	SocketMode  os.FileMode
}

// Serve menjalankan helper produksi. Helper produksi hanya boleh menyentuh
// file chap-secrets router yang telah ditentukan.
func Serve(ctx context.Context, socketPath, secretFile, chapSecrets string) error {
	return ServeWithConfig(ctx, ServeConfig{SocketPath: socketPath, SecretFile: secretFile, ChapSecrets: chapSecrets, SocketUID: -1, SocketGID: -1, SocketMode: 0o660})
}

func ServeWithConfig(ctx context.Context, config ServeConfig) error {
	if err := validateHelperConfig(config); err != nil {
		return err
	}
	secret, err := readSecret(config.SecretFile)
	if err != nil {
		return err
	}
	if err := prepareSocketPath(config.SocketPath); err != nil {
		return err
	}
	listener, err := net.Listen("unix", config.SocketPath)
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close(); _ = removeSocketPath(config.SocketPath) }()
	mode := config.SocketMode
	if mode == 0 {
		mode = 0o660
	}
	if err := os.Chmod(config.SocketPath, mode); err != nil {
		_ = listener.Close()
		return err
	}
	if config.SocketUID >= 0 || config.SocketGID >= 0 {
		uid, gid := config.SocketUID, config.SocketGID
		if uid < 0 {
			uid = os.Geteuid()
		}
		if gid < 0 {
			gid = os.Getegid()
		}
		if err := os.Chown(config.SocketPath, uid, gid); err != nil {
			_ = listener.Close()
			return err
		}
	}
	store := NewFileStore(config.ChapSecrets)
	replay := newReplayCache()
	sem := make(chan struct{}, 16)
	for {
		_ = listener.(*net.UnixListener).SetDeadline(time.Now().Add(time.Second))
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			continue
		}
		select {
		case sem <- struct{}{}:
			go func() { defer func() { <-sem }(); serveConn(conn, secret, store, replay) }()
		default:
			_ = conn.Close()
		}
	}
}

func serveConn(conn net.Conn, secret []byte, store *FileStore, replay *replayCache) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	var req helperRequest
	limited := &ioLimitConn{Conn: conn, Remaining: helperRequestLimit}
	if err := json.NewDecoder(bufio.NewReader(limited)).Decode(&req); err != nil || !validRequestMAC(secret, req) || !validRequestTime(req.Timestamp) || !replay.Accept(req.Nonce, time.Now()) {
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

func validRequestMAC(secret []byte, req helperRequest) bool {
	expected, err := hex.DecodeString(req.MAC)
	if err != nil {
		return false
	}
	actual, err := hex.DecodeString(signRequest(secret, req))
	if err != nil {
		return false
	}
	return hmac.Equal(expected, actual)
}
func validRequestTime(timestamp int64) bool {
	now := time.Now().UTC()
	requestTime := time.Unix(timestamp, 0)
	return timestamp > 0 && requestTime.After(now.Add(-helperMaxAge)) && requestTime.Before(now.Add(helperMaxAge))
}

type replayCache struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

func newReplayCache() *replayCache { return &replayCache{entries: map[string]time.Time{}} }
func (c *replayCache) Accept(nonce string, now time.Time) bool {
	if len(nonce) != 32 {
		return false
	}
	if _, err := hex.DecodeString(nonce); err != nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, expiry := range c.entries {
		if !expiry.After(now) {
			delete(c.entries, key)
		}
	}
	if _, exists := c.entries[nonce]; exists || len(c.entries) >= helperMaxNonces {
		return false
	}
	c.entries[nonce] = now.Add(helperMaxAge)
	return true
}

func validateHelperConfig(config ServeConfig) error {
	if config.SocketPath == "" || config.SecretFile == "" || config.ChapSecrets == "" {
		return errors.New("konfigurasi helper tidak lengkap")
	}
	chap := filepath.Clean(config.ChapSecrets)
	if config.Fixture {
		if os.Geteuid() == 0 {
			return errors.New("mode fixture tidak boleh berjalan sebagai root")
		}
		root := filepath.Clean("/var/lib/codenet-v04/helper-test")
		if chap != root && !strings.HasPrefix(chap, root+string(os.PathSeparator)) {
			return errors.New("fixture harus berada di bawah /var/lib/codenet-v04/helper-test")
		}
	} else if chap != "/etc/codenet/chap-secrets" {
		return errors.New("helper produksi hanya mengizinkan /etc/codenet/chap-secrets")
	}
	return nil
}

func prepareSocketPath(path string) error {
	parent := filepath.Dir(path)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&0o022 != 0 || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("direktori socket helper tidak aman")
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeSocket == 0 || !ownedByCurrentUser(info) {
			return errors.New("path socket helper tidak aman")
		}
		return removeSocketPath(path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func removeSocketPath(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeSocket == 0 || !ownedByCurrentUser(info) {
		return errors.New("socket helper bukan socket milik helper")
	}
	return os.Remove(path)
}
