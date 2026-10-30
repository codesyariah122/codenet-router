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
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
)

// LinuxProvider hanya menjalankan operasi pembacaan yang telah ditentukan.
// Tidak ada input pengguna yang diteruskan sebagai shell command.
type LinuxProvider struct{}

func NewLinux() *LinuxProvider { return &LinuxProvider{} }

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
	// Nilai CPU adalah snapshot load rata-rata 1 menit yang dinormalisasi terhadap jumlah CPU.
	load, err := readLimitedFile(ctx, "/proc/loadavg", 4096)
	if err != nil {
		return models.ResourceUsage{}, err
	}
	loadOne, _ := strconv.ParseFloat(strings.Fields(string(load))[0], 64)
	cpuCount := 1
	if data, e := readLimitedFile(ctx, "/proc/cpuinfo", 4*1024*1024); e == nil {
		cpuCount = strings.Count(string(data), "processor\t:")
		if cpuCount < 1 {
			cpuCount = 1
		}
	}
	cpuPercent := loadOne / float64(cpuCount) * 100
	if cpuPercent > 100 {
		cpuPercent = 100
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
	sessions, err := p.PPPoESessions(ctx)
	if err != nil {
		return models.PPPoEStatus{}, err
	}
	return models.PPPoEStatus{ServiceName: "CODENET-PPPOE", AccessConcentrator: "CODENET", Status: status, ActiveSessions: len(sessions), Pool: "10.60.0.10-10.60.0.50"}, nil
}

func (p *LinuxProvider) PPPoESessions(context.Context) ([]models.PPPoESession, error) {
	// Integrasi CLI Accel-PPP sengaja belum diaktifkan sampai socket kontrol dan format output dikonfigurasi eksplisit.
	return []models.PPPoESession{}, nil
}

func (p *LinuxProvider) BandwidthProfiles(context.Context) ([]models.BandwidthProfile, error) {
	return []models.BandwidthProfile{{Name: "default", DownloadBPS: 10_000_000, UploadBPS: 5_000_000, Description: "Nilai konfigurasi CODENET saat ini"}}, nil
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
		stats = append(stats, models.BandwidthStatistic{Interface: item.Name, DownloadLimit: 10_000_000, UploadLimit: 5_000_000, LimiterStatus: limiter})
	}
	return stats, nil
}

func (p *LinuxProvider) TrafficHistory(ctx context.Context) ([]models.TrafficPoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []models.TrafficPoint{}, nil
}
