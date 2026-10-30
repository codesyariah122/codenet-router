//go:build linux

package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
)

func (p *LinuxProvider) IPAddresses(ctx context.Context) ([]models.IPAddress, error) {
	links, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("membaca interface: %w", err)
	}
	addresses := make([]models.IPAddress, 0)
	for _, link := range links {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		addrs, err := link.Addrs()
		if err != nil {
			return nil, fmt.Errorf("membaca alamat interface %s: %w", link.Name, err)
		}
		for _, raw := range addrs {
			ip, network, err := net.ParseCIDR(raw.String())
			if err != nil {
				continue
			}
			version := 4
			if ip.To4() == nil {
				version = 6
			}
			flags := []string{}
			if link.Flags&net.FlagUp != 0 {
				flags = append(flags, "up")
			} else {
				flags = append(flags, "down")
			}
			ones, _ := network.Mask.Size()
			addresses = append(addresses, models.IPAddress{
				Address: ip.String() + "/" + strconv.Itoa(ones),
				Network: network.String(), Interface: link.Name, Version: version,
				Scope: ipScope(ip), Flags: flags,
			})
		}
	}
	return addresses, nil
}

func ipScope(ip net.IP) string {
	switch {
	case ip.IsLoopback():
		return "host"
	case ip.IsLinkLocalUnicast():
		return "link"
	case ip.IsPrivate():
		return "site"
	default:
		return "global"
	}
}

type ipRouteJSON struct {
	Dst      string          `json:"dst"`
	Gateway  string          `json:"gateway"`
	Dev      string          `json:"dev"`
	Metric   json.RawMessage `json:"metric"`
	Protocol string          `json:"protocol"`
	Scope    string          `json:"scope"`
	Table    json.RawMessage `json:"table"`
	Type     string          `json:"type"`
}

func (p *LinuxProvider) Routes(ctx context.Context) ([]models.Route, error) {
	v4, err := readIPRoutes(ctx, 4)
	if err != nil {
		return nil, err
	}
	v6, err := readIPRoutes(ctx, 6)
	if err != nil {
		return nil, err
	}
	return append(v4, v6...), nil
}

func readIPRoutes(ctx context.Context, version int) ([]models.Route, error) {
	if version != 4 && version != 6 {
		return nil, fmt.Errorf("versi IP tidak valid")
	}
	args := []string{"-j", fmt.Sprintf("-%d", version), "route", "show", "table", "all"}
	cmdCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var output limitedBuffer
	output.max = 2 * 1024 * 1024
	command := exec.CommandContext(cmdCtx, existingIPBinary(), args...)
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		if cmdCtx.Err() != nil {
			return nil, cmdCtx.Err()
		}
		return nil, fmt.Errorf("membaca routing IPv%d: %w", version, err)
	}
	return parseIPRoutes(output.Bytes(), version)
}

func existingIPBinary() string {
	for _, path := range []string{"/usr/sbin/ip", "/sbin/ip"} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "/usr/sbin/ip"
}

func parseIPRoutes(data []byte, version int) ([]models.Route, error) {
	var entries []ipRouteJSON
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("mem-parsing output ip: %w", err)
	}
	routes := make([]models.Route, 0, len(entries))
	for _, entry := range entries {
		destination := entry.Dst
		if destination == "" {
			destination = "default"
			if version == 6 {
				destination = "::/0"
			}
		}
		table := routeString(entry.Table, "main")
		protocol := entry.Protocol
		if protocol == "" {
			protocol = "unknown"
		}
		routeType := entry.Type
		if routeType == "" {
			routeType = "unicast"
		}
		status := "active"
		routes = append(routes, models.Route{Destination: destination, Gateway: entry.Gateway, Interface: entry.Dev, Metric: routeUint(entry.Metric), Protocol: protocol, Scope: entry.Scope, Table: table, Version: version, Type: routeType, Status: status})
	}
	return routes, nil
}

func routeString(raw json.RawMessage, fallback string) string {
	if len(raw) == 0 || string(raw) == "null" {
		return fallback
	}
	var text string
	if json.Unmarshal(raw, &text) == nil && text != "" {
		return text
	}
	var number uint32
	if json.Unmarshal(raw, &number) == nil {
		return strconv.FormatUint(uint64(number), 10)
	}
	return fallback
}

func routeUint(raw json.RawMessage) uint32 {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var number uint32
	if json.Unmarshal(raw, &number) == nil {
		return number
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		value, _ := strconv.ParseUint(strings.TrimSpace(text), 10, 32)
		return uint32(value)
	}
	return 0
}
