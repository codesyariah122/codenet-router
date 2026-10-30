package models

import "time"

type SystemInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Version  string `json:"version"`
	Kernel   string `json:"kernel"`
	Uptime   int64  `json:"uptime_seconds"`
}

type ResourceUsage struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryUsed    uint64  `json:"memory_used_bytes"`
	MemoryTotal   uint64  `json:"memory_total_bytes"`
}

type ServiceStatus struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
}

type Interface struct {
	Name      string   `json:"name"`
	Role      string   `json:"role"`
	Type      string   `json:"type"`
	MAC       string   `json:"mac_address"`
	IPv4      []string `json:"ipv4_addresses"`
	Status    string   `json:"status"`
	MTU       int      `json:"mtu"`
	RXBytes   uint64   `json:"rx_bytes"`
	TXBytes   uint64   `json:"tx_bytes"`
	RXPackets uint64   `json:"rx_packets"`
	TXPackets uint64   `json:"tx_packets"`
}

type IPAddress struct {
	Address   string   `json:"address"`
	Network   string   `json:"network"`
	Interface string   `json:"interface"`
	Version   int      `json:"ip_version"`
	Scope     string   `json:"scope"`
	Flags     []string `json:"flags"`
}

type Route struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway,omitempty"`
	Interface   string `json:"interface,omitempty"`
	Metric      uint32 `json:"metric,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Table       string `json:"routing_table,omitempty"`
	Version     int    `json:"ip_version"`
	Type        string `json:"route_type"`
	Status      string `json:"status"`
}

type PPPoEStatus struct {
	ServiceName        string `json:"service_name"`
	AccessConcentrator string `json:"access_concentrator"`
	ServerInterface    string `json:"server_interface,omitempty"`
	Status             string `json:"status"`
	ActiveSessions     int    `json:"active_sessions"`
	SessionCountReady  bool   `json:"session_count_ready"`
	Pool               string `json:"ip_pool"`
	PoolSource         string `json:"pool_source,omitempty"`
	ControlStatus      string `json:"control_status"`
	MonitoringError    string `json:"monitoring_error,omitempty"`
}

type PPPoESession struct {
	Username      string    `json:"username"`
	IP            string    `json:"ip_address"`
	Interface     string    `json:"interface"`
	ConnectedAt   time.Time `json:"connected_at"`
	UptimeSeconds int64     `json:"uptime_seconds"`
	RXBytes       uint64    `json:"rx_bytes"`
	TXBytes       uint64    `json:"tx_bytes"`
	DownloadBPS   uint64    `json:"download_bps"`
	UploadBPS     uint64    `json:"upload_bps"`
	Status        string    `json:"status"`
}

type BandwidthProfile struct {
	Name        string `json:"name"`
	DownloadBPS uint64 `json:"download_bps"`
	UploadBPS   uint64 `json:"upload_bps"`
	Description string `json:"description,omitempty"`
}

type BandwidthStatistic struct {
	Username      string `json:"username"`
	IP            string `json:"ip_address"`
	Interface     string `json:"interface"`
	DownloadLimit uint64 `json:"download_limit_bps"`
	UploadLimit   uint64 `json:"upload_limit_bps"`
	DownloadUsage uint64 `json:"download_usage_bps"`
	UploadUsage   uint64 `json:"upload_usage_bps"`
	LimiterStatus string `json:"limiter_status"`
}

type TrafficPoint struct {
	Timestamp time.Time `json:"timestamp"`
	RXBPS     uint64    `json:"rx_bps"`
	TXBPS     uint64    `json:"tx_bps"`
}

type Dashboard struct {
	System         SystemInfo     `json:"system"`
	Resources      ResourceUsage  `json:"resources"`
	Interfaces     []Interface    `json:"interfaces"`
	PPPoE          PPPoEStatus    `json:"pppoe"`
	TrafficHistory []TrafficPoint `json:"traffic_history"`
}
