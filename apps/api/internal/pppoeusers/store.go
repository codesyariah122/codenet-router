package pppoeusers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
)

var (
	ErrUnavailable = errors.New("PPPoE user helper tidak tersedia")
	ErrInvalid     = errors.New("data PPPoE user tidak valid")
	ErrConflict    = errors.New("username PPPoE sudah ada")
	ErrNotFound    = errors.New("username PPPoE tidak ditemukan")
)

type Manager interface {
	List(context.Context) ([]models.PPPoEUser, error)
	Create(context.Context, UserInput) (models.PPPoEUser, error)
	Update(context.Context, string, UserInput) (models.PPPoEUser, error)
	SetEnabled(context.Context, string, bool) (models.PPPoEUser, error)
	Delete(context.Context, string) error
}

type UserInput struct {
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	Server      string `json:"server"`
	IPPool      string `json:"ip_pool"`
	DownloadBPS uint64 `json:"download_bps"`
	UploadBPS   uint64 `json:"upload_bps"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

type AuditEntry struct {
	At        time.Time `json:"at"`
	Operation string    `json:"operation"`
	Username  string    `json:"username"`
	Success   bool      `json:"success"`
}

type UnavailableManager struct{}

func NewUnavailableManager() Manager { return UnavailableManager{} }
func (UnavailableManager) List(context.Context) ([]models.PPPoEUser, error) {
	return nil, ErrUnavailable
}
func (UnavailableManager) Create(context.Context, UserInput) (models.PPPoEUser, error) {
	return models.PPPoEUser{}, ErrUnavailable
}
func (UnavailableManager) Update(context.Context, string, UserInput) (models.PPPoEUser, error) {
	return models.PPPoEUser{}, ErrUnavailable
}
func (UnavailableManager) SetEnabled(context.Context, string, bool) (models.PPPoEUser, error) {
	return models.PPPoEUser{}, ErrUnavailable
}
func (UnavailableManager) Delete(context.Context, string) error { return ErrUnavailable }

// FileStore adalah backend fixture/helper. Backend ini hanya menulis file yang
// diberikan secara eksplisit dan tidak pernah diarahkan ke chap-secrets router
// dari HTTP. Helper produksi harus menjalankannya sebagai proses terpisah.
type FileStore struct {
	Path      string
	AuditPath string
	LockPath  string
	StatePath string
	MaxBytes  int64
}

func (s *FileStore) limit() int64 {
	if s.MaxBytes <= 0 {
		return 1 << 20
	}
	return s.MaxBytes
}

func NewFileStore(path string) *FileStore {
	return &FileStore{Path: path, AuditPath: path + ".audit", LockPath: path + ".lock", StatePath: path + ".state", MaxBytes: 1 << 20}
}

func (s *FileStore) List(ctx context.Context) ([]models.PPPoEUser, error) {
	if err := validateStorePath(s.Path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return []models.PPPoEUser{}, nil
	}
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > s.limit() {
		return nil, ErrInvalid
	}
	users, err := parseChapSecrets(string(data))
	if err != nil {
		return nil, err
	}
	state, _ := readState(s.StatePath)
	for i := range users {
		if enabled, ok := state[users[i].Username]; ok && users[i].Enabled {
			users[i].Enabled = enabled
		}
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Username < users[j].Username })
	return users, nil
}

func (s *FileStore) Create(ctx context.Context, input UserInput) (models.PPPoEUser, error) {
	if err := ValidateInput(input, true); err != nil {
		return models.PPPoEUser{}, err
	}
	return s.mutate(ctx, "create", input.Username, func(users []models.PPPoEUser) ([]models.PPPoEUser, models.PPPoEUser, error) {
		for _, user := range users {
			if user.Username == input.Username {
				return nil, models.PPPoEUser{}, ErrConflict
			}
		}
		user := input.toModel(true)
		return append(users, user), user, nil
	})
}

func (s *FileStore) Update(ctx context.Context, username string, input UserInput) (models.PPPoEUser, error) {
	if err := ValidateUsername(username); err != nil {
		return models.PPPoEUser{}, err
	}
	input.Username = username
	if err := ValidateInput(input, false); err != nil {
		return models.PPPoEUser{}, err
	}
	if input.Password != "" && (len(input.Password) < 8 || len(input.Password) > 1024 || strings.ContainsAny(input.Password, "\r\n")) {
		return models.PPPoEUser{}, fmt.Errorf("%w: password harus 8-1024 karakter", ErrInvalid)
	}
	return s.mutate(ctx, "update", username, func(users []models.PPPoEUser) ([]models.PPPoEUser, models.PPPoEUser, error) {
		for i := range users {
			if users[i].Username == username {
				updated := users[i]
				updated.Server, updated.IPPool = input.Server, input.IPPool
				updated.DownloadBPS, updated.UploadBPS = input.DownloadBPS, input.UploadBPS
				if input.Password != "" {
					updated.Password = input.Password
				}
				if input.Enabled != nil {
					updated.Enabled = *input.Enabled
				}
				return users, updated, nil
			}
		}
		return nil, models.PPPoEUser{}, ErrNotFound
	})
}

func (s *FileStore) SetEnabled(ctx context.Context, username string, enabled bool) (models.PPPoEUser, error) {
	if err := ValidateUsername(username); err != nil {
		return models.PPPoEUser{}, err
	}
	return s.mutate(ctx, map[bool]string{true: "enable", false: "disable"}[enabled], username, func(users []models.PPPoEUser) ([]models.PPPoEUser, models.PPPoEUser, error) {
		for i := range users {
			if users[i].Username == username {
				users[i].Enabled = enabled
				return users, users[i], nil
			}
		}
		return nil, models.PPPoEUser{}, ErrNotFound
	})
}

func (s *FileStore) Delete(ctx context.Context, username string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	_, err := s.mutate(ctx, "delete", username, func(users []models.PPPoEUser) ([]models.PPPoEUser, models.PPPoEUser, error) {
		for i := range users {
			if users[i].Username == username {
				return append(users[:i], users[i+1:]...), users[i], nil
			}
		}
		return nil, models.PPPoEUser{}, ErrNotFound
	})
	return err
}

func (s *FileStore) mutate(ctx context.Context, operation, username string, fn func([]models.PPPoEUser) ([]models.PPPoEUser, models.PPPoEUser, error)) (models.PPPoEUser, error) {
	if err := validateStorePath(s.Path); err != nil {
		return models.PPPoEUser{}, err
	}
	if err := ctx.Err(); err != nil {
		return models.PPPoEUser{}, err
	}
	unlock, err := acquireLock(s.LockPath)
	if err != nil {
		return models.PPPoEUser{}, err
	}
	defer unlock()
	users, err := s.List(ctx)
	if err != nil {
		return models.PPPoEUser{}, err
	}
	updated, result, err := fn(users)
	if err != nil {
		s.audit(operation, username, false)
		return models.PPPoEUser{}, err
	}
	if err := s.atomicWrite(updated); err != nil {
		s.audit(operation, username, false)
		return models.PPPoEUser{}, err
	}
	if err := s.writeState(updated); err != nil {
		_ = restoreBackup(s.Path)
		s.audit(operation, username, false)
		return models.PPPoEUser{}, err
	}
	s.audit(operation, username, true)
	return result, nil
}

func restoreBackup(path string) error {
	data, err := os.ReadFile(path + ".bak")
	if err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if info, statErr := os.Stat(path + ".bak"); statErr == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(path, data, mode)
}

func (s *FileStore) atomicWrite(users []models.PPPoEUser) error {
	oldMode := os.FileMode(0o600)
	if info, err := os.Stat(s.Path); err == nil {
		oldMode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".codenet-chap-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(oldMode); err != nil {
		_ = tmp.Close()
		return err
	}
	content := serializeChapSecrets(users)
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if existing, err := os.ReadFile(s.Path); err == nil {
		if err := os.WriteFile(s.Path+".bak", existing, oldMode); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpName, s.Path); err != nil {
		return err
	}
	return nil
}

func (s *FileStore) audit(operation, username string, success bool) {
	entry := fmt.Sprintf("%s operation=%s username=%s success=%t\n", time.Now().UTC().Format(time.RFC3339), operation, username, success)
	_ = os.MkdirAll(filepath.Dir(s.AuditPath), 0o700)
	f, err := os.OpenFile(s.AuditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err == nil {
		_, _ = f.WriteString(entry)
		_ = f.Close()
	}
}

func parseChapSecrets(content string) ([]models.PPPoEUser, error) {
	users := []models.PPPoEUser{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		disabled := false
		if strings.HasPrefix(line, "# codenet-disabled ") {
			disabled = true
			line = strings.TrimSpace(strings.TrimPrefix(line, "# codenet-disabled "))
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 5 {
			return nil, fmt.Errorf("%w: chap-secrets harus 5 kolom", ErrInvalid)
		}
		if err := ValidateUsername(fields[0]); err != nil {
			return nil, err
		}
		if fields[2] != "*" && !validServer(fields[2]) {
			return nil, ErrInvalid
		}
		if !validPool(fields[3]) {
			return nil, ErrInvalid
		}
		dl, ul, err := parseRate(fields[4])
		if err != nil {
			return nil, err
		}
		for _, existing := range users {
			if existing.Username == fields[0] {
				return nil, ErrConflict
			}
		}
		users = append(users, models.PPPoEUser{Username: fields[0], Server: fields[2], IPPool: fields[3], DownloadBPS: dl, UploadBPS: ul, Enabled: !disabled})
		users[len(users)-1].Password = fields[1]
	}
	return users, scanner.Err()
}

func serializeChapSecrets(users []models.PPPoEUser) string {
	var b strings.Builder
	for _, user := range users {
		// Password tidak dapat dipulihkan dari model read-only; helper produksi
		// harus mempertahankan secret lama atau menerima password baru secara aman.
		line := user.Username + " " + user.Password + " " + user.Server + " " + user.IPPool + " " + formatRate(user.DownloadBPS, user.UploadBPS) + "\n"
		if !user.Enabled {
			line = "# codenet-disabled " + line
		}
		b.WriteString(line)
	}
	return b.String()
}

func ValidateInput(input UserInput, creating bool) error {
	if err := ValidateUsername(input.Username); err != nil {
		return err
	}
	if creating && (len(input.Password) < 8 || len(input.Password) > 1024 || strings.ContainsAny(input.Password, "\r\n")) {
		return fmt.Errorf("%w: password harus 8-1024 karakter", ErrInvalid)
	}
	if !validServer(input.Server) || !validPool(input.IPPool) {
		return fmt.Errorf("%w: server atau IP pool tidak valid", ErrInvalid)
	}
	if input.DownloadBPS == 0 || input.UploadBPS == 0 || input.DownloadBPS > 1_000_000_000 || input.UploadBPS > 1_000_000_000 {
		return fmt.Errorf("%w: rate limit tidak valid", ErrInvalid)
	}
	return nil
}

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
var poolPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`)

func ValidateUsername(value string) error {
	if !usernamePattern.MatchString(value) {
		return fmt.Errorf("%w: username tidak valid", ErrInvalid)
	}
	return nil
}
func validServer(value string) bool { return value == "*" || poolPattern.MatchString(value) }
func validPool(value string) bool   { return poolPattern.MatchString(value) }
func parseRate(value string) (uint64, uint64, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return 0, 0, ErrInvalid
	}
	dl, e1 := strconv.ParseUint(parts[0], 10, 64)
	ul, e2 := strconv.ParseUint(parts[1], 10, 64)
	if e1 != nil || e2 != nil || dl == 0 || ul == 0 || dl > 1_000_000_000 || ul > 1_000_000_000 {
		return 0, 0, ErrInvalid
	}
	return dl, ul, nil
}
func formatRate(dl, ul uint64) string {
	return strconv.FormatUint(dl, 10) + "/" + strconv.FormatUint(ul, 10)
}

func acquireLock(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("memperoleh lock chap-secrets: %w", err)
	}
	_ = f.Close()
	return func() { _ = os.Remove(path) }, nil
}

func validateStorePath(path string) error {
	if path == "" || filepath.Base(path) == "." || filepath.Base(path) == ".." || strings.Contains(path, "\x00") {
		return ErrInvalid
	}
	return nil
}

func (u UserInput) toModel(enabled bool) models.PPPoEUser {
	return models.PPPoEUser{Username: u.Username, Server: u.Server, IPPool: u.IPPool, DownloadBPS: u.DownloadBPS, UploadBPS: u.UploadBPS, Enabled: enabled, Password: u.Password}
}

func (s *FileStore) writeState(users []models.PPPoEUser) error {
	state := map[string]bool{}
	for _, user := range users {
		state[user.Username] = user.Enabled
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.StatePath), ".codenet-state-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, s.StatePath)
}

func readState(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]bool{}, err
	}
	state := map[string]bool{}
	if err := json.Unmarshal(data, &state); err != nil {
		return map[string]bool{}, err
	}
	return state, nil
}
