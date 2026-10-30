package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
)

var (
	ErrAccelPPPControlUnavailable = errors.New("Accel-PPP control interface tidak tersedia")
	ErrAccelPPPInvalidResponse    = errors.New("respons control interface Accel-PPP tidak valid")
	ErrAccelPPPAuthentication     = errors.New("autentikasi CLI Accel-PPP gagal")
	ErrAccelPPPTimeout            = errors.New("timeout CLI Accel-PPP")
)

type MonitoringUnavailableError struct{ Cause error }

func (e MonitoringUnavailableError) Error() string {
	if e.Cause == nil {
		return ErrAccelPPPControlUnavailable.Error()
	}
	return e.Cause.Error()
}

func (e MonitoringUnavailableError) Unwrap() error { return e.Cause }

type AccelPPPInfo struct {
	AccessConcentrator string `json:"access_concentrator"`
	ServerInterface    string `json:"server_interface"`
	Pool               string `json:"ip_pool"`
	PoolSource         string `json:"pool_source"`
}

type AccelPPPMonitor interface {
	Info(context.Context) (AccelPPPInfo, error)
	Sessions(context.Context) ([]models.PPPoESession, error)
}

type unavailableAccelPPPMonitor struct{}

func NewUnavailableAccelPPPMonitor() AccelPPPMonitor { return unavailableAccelPPPMonitor{} }

func (unavailableAccelPPPMonitor) Info(context.Context) (AccelPPPInfo, error) {
	return AccelPPPInfo{}, MonitoringUnavailableError{Cause: ErrAccelPPPControlUnavailable}
}

func (unavailableAccelPPPMonitor) Sessions(context.Context) ([]models.PPPoESession, error) {
	return nil, MonitoringUnavailableError{Cause: ErrAccelPPPControlUnavailable}
}

// TCPAccelPPPMonitor berbicara langsung dengan CLI TCP Accel-PPP. Adapter ini
// hanya mengirim perintah monitoring tetap dan tidak menerima command dari
// HTTP request.
type TCPAccelPPPMonitor struct {
	Host         string
	Port         string
	PasswordFile string
	Timeout      time.Duration
	dial         func(context.Context, string, string) (net.Conn, error)
}

func NewTCPAccelPPPMonitor(host, port, passwordFile string) *TCPAccelPPPMonitor {
	return &TCPAccelPPPMonitor{Host: host, Port: port, PasswordFile: passwordFile, Timeout: 3 * time.Second, dial: (&net.Dialer{}).DialContext}
}

func (m *TCPAccelPPPMonitor) Info(ctx context.Context) (AccelPPPInfo, error) {
	if _, err := m.query(ctx); err != nil {
		return AccelPPPInfo{}, err
	}
	// Accel-PPP CLI "show sessions" tidak selalu memuat AC/interface/pool.
	// Jangan mengisi nilai yang tidak diverifikasi; status control tetap valid.
	return AccelPPPInfo{PoolSource: "tidak tersedia dari CLI"}, nil
}

func (m *TCPAccelPPPMonitor) Sessions(ctx context.Context) ([]models.PPPoESession, error) {
	response, err := m.query(ctx)
	if err != nil {
		return nil, err
	}
	sessions, err := parseAccelPPPSessions(response)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (m *TCPAccelPPPMonitor) query(ctx context.Context) ([]byte, error) {
	if m == nil || strings.TrimSpace(m.Host) == "" || strings.TrimSpace(m.Port) == "" || strings.TrimSpace(m.PasswordFile) == "" {
		return nil, MonitoringUnavailableError{Cause: ErrAccelPPPControlUnavailable}
	}
	port, portErr := strconv.Atoi(strings.TrimSpace(m.Port))
	if portErr != nil || port < 1 || port > 65535 {
		return nil, MonitoringUnavailableError{Cause: errors.New("port CLI Accel-PPP tidak valid")}
	}
	password, err := readAccelPPPPassword(m.PasswordFile)
	if err != nil {
		return nil, MonitoringUnavailableError{Cause: err}
	}
	timeout := m.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dial := m.dial
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	conn, err := dial(callCtx, "tcp", net.JoinHostPort(m.Host, strconv.Itoa(port)))
	if err != nil {
		if errors.Is(callCtx.Err(), context.DeadlineExceeded) {
			return nil, MonitoringUnavailableError{Cause: ErrAccelPPPTimeout}
		}
		return nil, MonitoringUnavailableError{Cause: err}
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	// accel-cmd menggunakan protokol line-oriented: password, command, lalu
	// exit agar daemon menutup sesi CLI setelah respons dikirim.
	// Mengirim credential melalui socket menghindari password terlihat pada argv.
	if _, err := io.WriteString(conn, password+"\nshow sessions\nexit\n"); err != nil {
		return nil, MonitoringUnavailableError{Cause: err}
	}
	output, err := readCLIOutput(conn, 2*1024*1024)
	if err != nil {
		return nil, err
	}
	lower := strings.ToLower(string(output))
	if strings.Contains(lower, "invalid password") || strings.Contains(lower, "authentication failed") || strings.Contains(lower, "access denied") || strings.Contains(lower, "wrong password") {
		return nil, MonitoringUnavailableError{Cause: ErrAccelPPPAuthentication}
	}
	if strings.Contains(lower, "password:") && !strings.Contains(lower, "ifname") {
		return nil, MonitoringUnavailableError{Cause: ErrAccelPPPAuthentication}
	}
	return output, nil
}

func readAccelPPPPassword(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("membaca file password Accel-PPP: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return "", errors.New("permission file password Accel-PPP harus owner-only")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("membaca file password Accel-PPP: %w", err)
	}
	if len(data) > 4096 {
		return "", errors.New("file password Accel-PPP terlalu besar")
	}
	password := strings.TrimSpace(string(data))
	if password == "" || strings.ContainsAny(password, "\r\n") {
		return "", errors.New("password Accel-PPP kosong atau tidak valid")
	}
	return password, nil
}

func readCLIOutput(conn net.Conn, max int) ([]byte, error) {
	var output []byte
	buf := make([]byte, 4096)
	for len(output) < max {
		n, err := conn.Read(buf)
		if n > 0 {
			if len(output)+n > max {
				return nil, ErrAccelPPPInvalidResponse
			}
			output = append(output, buf[:n]...)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return output, nil
			}
			if errors.Is(err, os.ErrDeadlineExceeded) || strings.Contains(strings.ToLower(err.Error()), "i/o timeout") {
				return nil, MonitoringUnavailableError{Cause: ErrAccelPPPTimeout}
			}
			return nil, MonitoringUnavailableError{Cause: err}
		}
	}
	return nil, ErrAccelPPPInvalidResponse
}

func hasCLIPrompt(output []byte) bool {
	for _, line := range strings.Split(strings.ReplaceAll(string(output), "\r", ""), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && len(line) < 80 && (strings.HasSuffix(line, ">") || strings.HasSuffix(line, "#")) {
			return true
		}
	}
	return false
}

func parseAccelPPPSessions(raw []byte) ([]models.PPPoESession, error) {
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r", ""), "\n")
	header := -1
	pipe := false
	for i, line := range lines {
		low := strings.ToLower(line)
		if strings.Contains(low, "ifname") && strings.Contains(low, "username") && strings.Contains(low, "uptime") {
			header = i
			pipe = strings.Contains(line, "|")
			break
		}
	}
	if header < 0 {
		return nil, fmt.Errorf("%w: header sessions tidak ditemukan", ErrAccelPPPInvalidResponse)
	}
	result := make([]models.PPPoESession, 0)
	sawTableMarker := false
	for _, line := range lines[header+1:] {
		line = strings.TrimSpace(line)
		if line == "" || hasCLIPrompt([]byte(line)) {
			continue
		}
		if isSeparator(line) {
			sawTableMarker = true
			continue
		}
		var fields []string
		if pipe {
			fields = splitPipeFields(line)
		} else {
			fields = strings.Fields(line)
		}
		if len(fields) < 9 {
			return nil, fmt.Errorf("%w: baris sessions kurang kolom", ErrAccelPPPInvalidResponse)
		}
		uptime, err := parseAccelPPPUptime(fields[8])
		if err != nil {
			return nil, fmt.Errorf("%w: uptime: %v", ErrAccelPPPInvalidResponse, err)
		}
		session := models.PPPoESession{Interface: fields[0], Username: fields[1], IP: fields[3], UptimeSeconds: uptime, ConnectedAt: time.Now().Add(-time.Duration(uptime) * time.Second), Status: fields[7]}
		if err := validateSession(session); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAccelPPPInvalidResponse, err)
		}
		sawTableMarker = true
		result = append(result, session)
		if len(result) > 1024 {
			return nil, ErrAccelPPPInvalidResponse
		}
	}
	if !sawTableMarker {
		return nil, fmt.Errorf("%w: tabel sessions terpotong", ErrAccelPPPInvalidResponse)
	}
	return result, nil
}

func splitPipeFields(line string) []string {
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func isSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	for _, r := range trimmed {
		if !strings.ContainsRune("-|+:= ", r) {
			return false
		}
	}
	return true
}

func parseAccelPPPUptime(value string) (int64, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, errors.New("format uptime harus HH:MM:SS")
	}
	h, err1 := strconv.ParseInt(parts[0], 10, 32)
	m, err2 := strconv.ParseInt(parts[1], 10, 32)
	s, err3 := strconv.ParseInt(parts[2], 10, 32)
	if err1 != nil || err2 != nil || err3 != nil || h < 0 || m < 0 || m >= 60 || s < 0 || s >= 60 {
		return 0, errors.New("nilai uptime tidak valid")
	}
	return h*3600 + m*60 + s, nil
}

// JSONSocketAccelPPPMonitor adalah adapter kontrak lokal yang sengaja tidak
// diaktifkan tanpa CODENET_ACCEL_PPP_CONTROL_SOCKET. Format ini harus
// disediakan oleh helper/control adapter yang telah diverifikasi pada Debian.
type JSONSocketAccelPPPMonitor struct {
	Path    string
	Timeout time.Duration
	dial    func(context.Context, string, string) (net.Conn, error)
}

func NewJSONSocketAccelPPPMonitor(path string) *JSONSocketAccelPPPMonitor {
	return &JSONSocketAccelPPPMonitor{Path: path, Timeout: 2 * time.Second, dial: (&net.Dialer{}).DialContext}
}

func (m *JSONSocketAccelPPPMonitor) Info(ctx context.Context) (AccelPPPInfo, error) {
	var response struct {
		Success bool         `json:"success"`
		Data    AccelPPPInfo `json:"data"`
		Error   string       `json:"error"`
	}
	if err := m.request(ctx, "status", &response); err != nil {
		return AccelPPPInfo{}, err
	}
	if !response.Success {
		return AccelPPPInfo{}, fmt.Errorf("control status gagal: %s", response.Error)
	}
	return response.Data, nil
}

func (m *JSONSocketAccelPPPMonitor) Sessions(ctx context.Context) ([]models.PPPoESession, error) {
	var response struct {
		Success bool                  `json:"success"`
		Data    []models.PPPoESession `json:"data"`
		Error   string                `json:"error"`
	}
	if err := m.request(ctx, "sessions", &response); err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, fmt.Errorf("control sessions gagal: %s", response.Error)
	}
	if len(response.Data) > 1024 {
		return nil, ErrAccelPPPInvalidResponse
	}
	for i := range response.Data {
		if err := validateSession(response.Data[i]); err != nil {
			return nil, fmt.Errorf("sesi %d tidak valid: %w", i, err)
		}
	}
	return response.Data, nil
}

func (m *JSONSocketAccelPPPMonitor) request(ctx context.Context, action string, target any) error {
	if m == nil || m.Path == "" {
		return MonitoringUnavailableError{Cause: ErrAccelPPPControlUnavailable}
	}
	timeout := m.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dial := m.dial
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	conn, err := dial(callCtx, "unix", m.Path)
	if err != nil {
		return MonitoringUnavailableError{Cause: err}
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	request, _ := json.Marshal(map[string]string{"action": action})
	if _, err := conn.Write(append(request, '\n')); err != nil {
		return MonitoringUnavailableError{Cause: err}
	}
	reader := bufio.NewReader(&ioLimitReader{reader: conn, remaining: 2 * 1024 * 1024})
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return MonitoringUnavailableError{Cause: err}
	}
	if len(line) > 2*1024*1024 {
		return ErrAccelPPPInvalidResponse
	}
	if err := json.Unmarshal(line, target); err != nil {
		return fmt.Errorf("%w: %v", ErrAccelPPPInvalidResponse, err)
	}
	return nil
}

type ioLimitReader struct {
	reader    net.Conn
	remaining int
}

func (r *ioLimitReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, errors.New("respons control interface terlalu besar")
	}
	if len(p) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.reader.Read(p)
	r.remaining -= n
	return n, err
}

var safeSessionInterface = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,15}$`)

func validateSession(session models.PPPoESession) error {
	if strings.TrimSpace(session.Username) == "" || len(session.Username) > 128 {
		return errors.New("username tidak valid")
	}
	if net.ParseIP(session.IP) == nil {
		return errors.New("alamat IP tidak valid")
	}
	if !safeSessionInterface.MatchString(session.Interface) {
		return errors.New("interface tidak valid")
	}
	if session.UptimeSeconds < 0 {
		return errors.New("uptime tidak valid")
	}
	return nil
}
