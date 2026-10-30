//go:build linux

package providers

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
)

// LinuxProvider hanya menjalankan operasi pembacaan yang telah ditentukan.
// Tidak ada input pengguna yang diteruskan sebagai shell command.
type LinuxProvider struct {
	mu             sync.Mutex
	lastSample     time.Time
	lastRXBytes    uint64
	lastTXBytes    uint64
	accelPPP       AccelPPPMonitor
	sessionMu      sync.Mutex
	sessionSamples map[string]sessionTrafficSample
}

type sessionTrafficSample struct {
	at time.Time
	rx uint64
	tx uint64
}

func NewLinux() *LinuxProvider {
	return NewLinuxWithAccelPPP(accelPPPMonitorFromEnv())
}

func NewLinuxWithAccelPPP(monitor AccelPPPMonitor) *LinuxProvider {
	if monitor == nil {
		monitor = NewUnavailableAccelPPPMonitor()
	}
	return &LinuxProvider{accelPPP: monitor, sessionSamples: make(map[string]sessionTrafficSample)}
}

func accelPPPMonitorFromEnv() AccelPPPMonitor {
	if passwordFile := strings.TrimSpace(os.Getenv("CODENET_ACCEL_PPP_CLI_PASSWORD_FILE")); passwordFile != "" {
		host := strings.TrimSpace(os.Getenv("CODENET_ACCEL_PPP_CLI_HOST"))
		if host == "" {
			host = "127.0.0.1"
		}
		port := strings.TrimSpace(os.Getenv("CODENET_ACCEL_PPP_CLI_PORT"))
		if port == "" {
			port = "2001"
		}
		return NewTCPAccelPPPMonitor(host, port, passwordFile)
	}
	if path := strings.TrimSpace(os.Getenv("CODENET_ACCEL_PPP_CONTROL_SOCKET")); path != "" {
		return NewJSONSocketAccelPPPMonitor(path)
	}
	return NewUnavailableAccelPPPMonitor()
}

func (p *LinuxProvider) SystemInfo(ctx context.Context) (models.SystemInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return models.SystemInfo{}, err
	}
	osName, version, err := readOSRelease(ctx)
	if err != nil {
		return models.SystemInfo{}, err
	}
	kernelBytes, err := readLimitedFile(ctx, "/proc/sys/kernel/osrelease", 4096)
	if err != nil {
		return models.SystemInfo{}, err
	}
	uptimeBytes, err := readLimitedFile(ctx, "/proc/uptime", 4096)
	if err != nil {
		return models.SystemInfo{}, err
	}
	uptime, _ := strconv.ParseFloat(strings.Fields(string(uptimeBytes))[0], 64)
	return models.SystemInfo{Hostname: hostname, OS: osName, Version: version, Kernel: strings.TrimSpace(string(kernelBytes)), Uptime: int64(uptime)}, nil
}

func readOSRelease(ctx context.Context) (string, string, error) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "", "", fmt.Errorf("membuka os-release: %w", err)
	}
	defer f.Close()
	values := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return "", "", err
		}
		parts := strings.SplitN(scanner.Text(), "=", 2)
		if len(parts) == 2 {
			values[parts[0]] = strings.Trim(parts[1], "\"")
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("membaca os-release: %w", err)
	}
	return values["PRETTY_NAME"], values["VERSION_ID"], nil
}

func (p *LinuxProvider) ResourceUsage(ctx context.Context) (models.ResourceUsage, error) {
	mem, err := readLimitedFile(ctx, "/proc/meminfo", 256*1024)
	if err != nil {
		return models.ResourceUsage{}, err
	}
	values := parseMemInfo(mem)
	total, available := values["MemTotal"], values["MemAvailable"]
	used := total - available
	memoryPercent := 0.0
	if total > 0 {
		memoryPercent = float64(used) / float64(total) * 100
	}
	// Utilisasi dihitung dari counter kumulatif kernel; bukan angka simulasi.
	cpuData, err := readLimitedFile(ctx, "/proc/stat", 64*1024)
	if err != nil {
		return models.ResourceUsage{}, err
	}
	cpuPercent := 0.0
	if busy, total, ok := parseCPUStat(cpuData); ok && total > 0 {
		cpuPercent = float64(busy) / float64(total) * 100
	}
	return models.ResourceUsage{CPUPercent: cpuPercent, MemoryPercent: memoryPercent, MemoryUsed: used, MemoryTotal: total}, nil
}

func (p *LinuxProvider) Services(ctx context.Context) ([]models.ServiceStatus, error) {
	names := []struct{ unit, display, description string }{{"codenet-pppoe.service", "Accel-PPP", "PPPoE access concentrator"}, {"nftables.service", "nftables", "Linux packet filtering"}, {"systemd-networkd.service", "Network", "Network interface management"}}
	result := make([]models.ServiceStatus, 0, len(names))
	for _, item := range names {
		status := "unknown"
		cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		var output limitedBuffer
		output.max = 64 * 1024
		command := exec.CommandContext(cmdCtx, "/usr/bin/systemctl", "is-active", item.unit)
		command.Stdout = &output
		command.Stderr = &output
		err := command.Run()
		cancel()
		if err == nil {
			status = strings.TrimSpace(output.String())
		} else if output.Len() > 0 {
			status = strings.TrimSpace(output.String())
		}
		result = append(result, models.ServiceStatus{Name: item.unit, DisplayName: item.display, Status: status, Description: item.description})
	}
	return result, nil
}

func (p *LinuxProvider) Interfaces(ctx context.Context) ([]models.Interface, error) {
	links, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	result := make([]models.Interface, 0, len(links))
	for _, link := range links {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if link.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := link.Addrs()
		if err != nil {
			return nil, fmt.Errorf("membaca alamat interface %s: %w", link.Name, err)
		}
		ipv4 := []string{}
		for _, addr := range addrs {
			if ip, _, e := net.ParseCIDR(addr.String()); e == nil && ip.To4() != nil {
				ipv4 = append(ipv4, addr.String())
			}
		}
		status := "down"
		if link.Flags&net.FlagUp != 0 {
			status = "up"
		}
		kind := "ethernet"
		role := "Other"
		if strings.HasPrefix(link.Name, "ppp") {
			kind, role = "ppp", "PPPoE"
		} else if link.Name == "enp0s3" {
			role = "WAN"
		} else if link.Name == "enp0s8" {
			role = "LAN"
		}
		stats, err := readInterfaceStats(ctx, link.Name)
		if err != nil {
			return nil, err
		}
		result = append(result, models.Interface{Name: link.Name, Role: role, Type: kind, MAC: link.HardwareAddr.String(), IPv4: ipv4, Status: status, MTU: link.MTU, RXBytes: stats[0], TXBytes: stats[1], RXPackets: stats[2], TXPackets: stats[3]})
	}
	return result, nil
}

func readInterfaceStats(ctx context.Context, name string) ([4]uint64, error) {
	keys := []string{"rx_bytes", "tx_bytes", "rx_packets", "tx_packets"}
	var out [4]uint64
	for i, key := range keys {
		data, err := readLimitedFile(ctx, "/sys/class/net/"+name+"/statistics/"+key, 4096)
		if err != nil {
			return out, fmt.Errorf("membaca statistik interface %s: %w", name, err)
		}
		out[i], err = strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		if err != nil {
			return out, fmt.Errorf("mem-parsing statistik interface %s: %w", name, err)
		}
	}
	return out, nil
}

func (p *LinuxProvider) Interface(ctx context.Context, name string) (models.Interface, error) {
	items, err := p.Interfaces(ctx)
	if err != nil {
		return models.Interface{}, err
	}
	for _, item := range items {
		if item.Name == name {
			return item, nil
		}
	}
	return models.Interface{}, NotFoundError{Resource: "interface " + name}
}

func (p *LinuxProvider) PPPoEStatus(ctx context.Context) (models.PPPoEStatus, error) {
	services, err := p.Services(ctx)
	if err != nil {
		return models.PPPoEStatus{}, err
	}
	status := "unknown"
	for _, service := range services {
		if service.Name == "codenet-pppoe.service" {
			status = service.Status
		}
	}
	result := models.PPPoEStatus{ServiceName: "CODENET-PPPOE", Status: status, ControlStatus: "unavailable", SessionCountReady: false}
	info, infoErr := p.accelPPP.Info(ctx)
	if infoErr != nil {
		result.MonitoringError = "Control interface Accel-PPP tidak tersedia"
		return result, nil
	}
	result.AccessConcentrator = info.AccessConcentrator
	result.ServerInterface = info.ServerInterface
	result.Pool = info.Pool
	result.PoolSource = info.PoolSource
	result.ControlStatus = "available"
	sessions, sessionsErr := p.PPPoESessions(ctx)
	if sessionsErr != nil {
		result.ControlStatus = "error"
		result.MonitoringError = "Sesi Accel-PPP tidak dapat dibaca"
		return result, nil
	}
	result.ActiveSessions = len(sessions)
	result.SessionCountReady = true
	return result, nil
}

func (p *LinuxProvider) PPPoESessions(ctx context.Context) ([]models.PPPoESession, error) {
	sessions, err := p.accelPPP.Sessions(ctx)
	if err != nil {
		return nil, MonitoringUnavailableError{Cause: err}
	}
	now := time.Now()
	for i := range sessions {
		stats, statsErr := readInterfaceStats(ctx, sessions[i].Interface)
		if statsErr != nil {
			sessions[i].Status = "interface_missing"
			sessions[i].DownloadBPS = 0
			sessions[i].UploadBPS = 0
			continue
		}
		sessions[i].RXBytes, sessions[i].TXBytes = stats[0], stats[1]
		p.sessionMu.Lock()
		previous, found := p.sessionSamples[sessions[i].Interface]
		p.sessionSamples[sessions[i].Interface] = sessionTrafficSample{at: now, rx: stats[0], tx: stats[1]}
		p.sessionMu.Unlock()
		if found {
			seconds := now.Sub(previous.at).Seconds()
			if seconds > 0 {
				sessions[i].UploadBPS = counterRate(previous.rx, stats[0], now.Sub(previous.at))
				sessions[i].DownloadBPS = counterRate(previous.tx, stats[1], now.Sub(previous.at))
			}
		}
	}
	return sessions, nil
}

func (p *LinuxProvider) BandwidthProfiles(context.Context) ([]models.BandwidthProfile, error) {
	// Profil Accel-PPP belum memiliki sumber read-only yang terverifikasi.
	return []models.BandwidthProfile{}, nil
}

func (p *LinuxProvider) BandwidthStatistics(ctx context.Context) ([]models.BandwidthStatistic, error) {
	// Daftar interface berasal dari kernel; tc hanya dipanggil dengan nama yang telah ditemukan, tanpa shell.
	interfaces, err := p.Interfaces(ctx)
	if err != nil {
		return nil, err
	}
	stats := []models.BandwidthStatistic{}
	for _, item := range interfaces {
		if item.Type != "ppp" {
			continue
		}
		cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		var output limitedBuffer
		output.max = 1024 * 1024
		command := exec.CommandContext(cmdCtx, "/usr/sbin/tc", "-s", "qdisc", "show", "dev", item.Name)
		command.Stdout = &output
		command.Stderr = &output
		cmdErr := command.Run()
		cancel()
		limiter := "unavailable"
		if cmdErr == nil && output.Len() > 0 {
			limiter = "active"
		}
		stats = append(stats, models.BandwidthStatistic{Interface: item.Name, LimiterStatus: limiter})
	}
	return stats, nil
}

func (p *LinuxProvider) TrafficHistory(ctx context.Context) ([]models.TrafficPoint, error) {
	interfaces, err := p.Interfaces(ctx)
	if err != nil {
		return nil, err
	}
	var rx, tx uint64
	for _, item := range interfaces {
		rx += item.RXBytes
		tx += item.TXBytes
	}
	now := time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()
	var rxRate, txRate uint64
	if !p.lastSample.IsZero() {
		seconds := now.Sub(p.lastSample).Seconds()
		if seconds > 0 && rx >= p.lastRXBytes && tx >= p.lastTXBytes {
			rxRate = uint64(float64(rx-p.lastRXBytes) / seconds)
			txRate = uint64(float64(tx-p.lastTXBytes) / seconds)
		}
	}
	p.lastSample, p.lastRXBytes, p.lastTXBytes = now, rx, tx
	return []models.TrafficPoint{{Timestamp: now, RXBPS: rxRate, TXBPS: txRate}}, nil
}
