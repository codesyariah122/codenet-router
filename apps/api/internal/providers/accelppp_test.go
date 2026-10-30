package providers

import (
	"bufio"
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
)

func TestJSONSocketAccelPPPMonitor(t *testing.T) {
	monitor := NewJSONSocketAccelPPPMonitor("test")
	var calls int
	monitor.dial = func(context.Context, string, string) (net.Conn, error) {
		client, server := net.Pipe()
		call := calls
		calls++
		go func() {
			defer server.Close()
			buf := make([]byte, 256)
			n, _ := server.Read(buf)
			if call == 0 && !strings.Contains(string(buf[:n]), `"sessions"`) {
				_, _ = server.Write([]byte(`{"success":true,"data":{"access_concentrator":"CODENET","server_interface":"enp0s8","ip_pool":"10.60.0.10-10.60.0.50","pool_source":"accel-ppp-control"}}` + "\n"))
			} else {
				_, _ = server.Write([]byte(`{"success":true,"data":[{"username":"alice","ip_address":"10.60.0.10","interface":"ppp0","uptime_seconds":12,"status":"online"}]}` + "\n"))
			}
		}()
		return client, nil
	}
	info, err := monitor.Info(context.Background())
	if err != nil || info.AccessConcentrator != "CODENET" {
		t.Fatalf("info=%#v err=%v", info, err)
	}
	sessions, err := monitor.Sessions(context.Background())
	if err != nil || len(sessions) != 1 || sessions[0].Username != "alice" {
		t.Fatalf("sessions=%#v err=%v", sessions, err)
	}
	if err := validateSession(models.PPPoESession{Username: "alice", IP: "invalid", Interface: "ppp0"}); err == nil {
		t.Fatal("expected invalid session validation")
	}
}

func TestUnavailableAccelPPPMonitor(t *testing.T) {
	monitor := NewUnavailableAccelPPPMonitor()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := monitor.Sessions(ctx); err == nil {
		t.Fatal("expected unavailable error")
	}
}

func TestJSONSocketAccelPPPMonitorMissingSocket(t *testing.T) {
	monitor := NewJSONSocketAccelPPPMonitor(filepath.Join(os.TempDir(), "missing-codenet-accel.sock"))
	if _, err := monitor.Info(context.Background()); err == nil {
		t.Fatal("expected missing socket error")
	}
}

func TestParseAccelPPPSessionsPipeTable(t *testing.T) {
	raw := []byte("ifname | username | calling-sid | ip | rate-limit | type | comp | state | uptime\n" +
		"-------+----------+-------------+----+------------+------+------+-" + "-----+-------\n" +
		"ppp0 | codenet-test01 | 08:00:27:63:99:62 | 10.60.0.10 | 10000/5000 | pppoe | | active | 00:00:23\n")
	sessions, err := parseAccelPPPSessions(raw)
	if err != nil || len(sessions) != 1 {
		t.Fatalf("sessions=%#v err=%v", sessions, err)
	}
	if sessions[0].Username != "codenet-test01" || sessions[0].UptimeSeconds != 23 || sessions[0].IP != "10.60.0.10" {
		t.Fatalf("unexpected session: %#v", sessions[0])
	}
}

func TestParseAccelPPPSessionsEmptyAndMalformed(t *testing.T) {
	empty, err := parseAccelPPPSessions([]byte("ifname | username | calling-sid | ip | rate-limit | type | comp | state | uptime\n" +
		"-------+----------+-------------+----+------------+------+------+-------+-------\n"))
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty=%#v err=%v", empty, err)
	}
	if _, err := parseAccelPPPSessions([]byte("ifname | username | calling-sid | ip | rate-limit | type | comp | state | uptime\n")); err == nil {
		t.Fatal("expected truncated table error")
	}
	if _, err := parseAccelPPPSessions([]byte("ifname | username | calling-sid | ip | rate-limit | type | comp | state | uptime\nppp0 | broken\n")); err == nil {
		t.Fatal("expected malformed response")
	}
}

func TestReadAccelPPPPasswordPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits tidak berlaku di Windows")
	}
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	password, err := readAccelPPPPassword(path)
	if err != nil || password != "secret" {
		t.Fatalf("password=%q err=%v", password, err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readAccelPPPPassword(path); err == nil {
		t.Fatal("expected insecure permission error")
	}
}

func TestTCPAccelPPPMonitorNetPipe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	monitor := NewTCPAccelPPPMonitor("127.0.0.1", "2001", path)
	monitor.dial = func(context.Context, string, string) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			reader := bufio.NewReader(server)
			var lines []string
			for len(lines) < 3 {
				line, readErr := reader.ReadString('\n')
				if readErr != nil {
					return
				}
				lines = append(lines, strings.TrimSpace(line))
			}
			if strings.Join(lines, "|") != "secret|show sessions|exit" {
				return
			}
			_, _ = server.Write([]byte("ifname | username | calling-sid | ip | rate-limit | type | comp | state | uptime\n" +
				"ppp0 | alice | sid | 10.60.0.10 | 10000/5000 | pppoe | | active | 00:01:02\n"))
		}()
		return client, nil
	}
	sessions, err := monitor.Sessions(context.Background())
	if err != nil || len(sessions) != 1 || sessions[0].Username != "alice" {
		t.Fatalf("sessions=%#v err=%v", sessions, err)
	}
}

func TestTCPAccelPPPMonitorMockTCPServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("environment tidak mengizinkan listener TCP lokal: %v", err)
	}
	defer listener.Close()
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		var lines []string
		for len(lines) < 3 {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				return
			}
			lines = append(lines, strings.TrimSpace(line))
		}
		if strings.Join(lines, "|") != "secret|show sessions|exit" {
			return
		}
		_, _ = conn.Write([]byte("ifname | username | calling-sid | ip | rate-limit | type | comp | state | uptime\n" +
			"ppp1 | bob | sid | 10.60.0.11 | 10000/5000 | pppoe | | active | 00:00:05\n"))
	}()
	addr := listener.Addr().(*net.TCPAddr)
	monitor := NewTCPAccelPPPMonitor("127.0.0.1", strconv.Itoa(addr.Port), path)
	sessions, err := monitor.Sessions(context.Background())
	if err != nil || len(sessions) != 1 || sessions[0].Username != "bob" {
		t.Fatalf("sessions=%#v err=%v", sessions, err)
	}
}

func TestTCPAccelPPPMonitorTerminatesCLIConversation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	monitor := NewTCPAccelPPPMonitor("127.0.0.1", "2001", path)
	monitor.Timeout = 250 * time.Millisecond
	monitor.dial = func(context.Context, string, string) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			reader := bufio.NewReader(server)
			var lines []string
			for len(lines) < 3 {
				line, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				lines = append(lines, strings.TrimSpace(line))
			}
			if strings.Join(lines, "|") != "secret|show sessions|exit" {
				return
			}
			_, _ = server.Write([]byte("ifname | username | calling-sid | ip | rate-limit | type | comp | state | uptime\n" +
				"-------+----------+-------------+----+------------+------+------+-------+-------\n"))
		}()
		return client, nil
	}
	sessions, err := monitor.Sessions(context.Background())
	if err != nil || len(sessions) != 0 {
		t.Fatalf("sessions=%#v err=%v", sessions, err)
	}
}

func TestTCPAccelPPPMonitorAuthenticationFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	monitor := NewTCPAccelPPPMonitor("127.0.0.1", "2001", path)
	monitor.dial = func(context.Context, string, string) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			buf := make([]byte, 128)
			_, _ = server.Read(buf)
			_, _ = server.Write([]byte("invalid password\n"))
		}()
		return client, nil
	}
	if _, err := monitor.Sessions(context.Background()); err == nil || !strings.Contains(strings.ToLower(err.Error()), "autentikasi") {
		t.Fatalf("expected authentication error, got %v", err)
	}
}

func TestTCPAccelPPPMonitorTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	monitor := NewTCPAccelPPPMonitor("127.0.0.1", "2001", path)
	monitor.Timeout = 20 * time.Millisecond
	monitor.dial = func(context.Context, string, string) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			buf := make([]byte, 128)
			_, _ = server.Read(buf)
			_, _ = server.Read(buf)
		}()
		return client, nil
	}
	if _, err := monitor.Sessions(context.Background()); err == nil || !strings.Contains(strings.ToLower(err.Error()), "timeout") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestTCPAccelPPPMonitorRejectsInvalidPort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	monitor := NewTCPAccelPPPMonitor("127.0.0.1", "not-a-port", path)
	if _, err := monitor.Sessions(context.Background()); err == nil {
		t.Fatal("expected invalid port error")
	}
}
