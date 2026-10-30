package providers

import (
	"context"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
)

type MockProvider struct {
	now func() time.Time
}

func NewMock() *MockProvider { return &MockProvider{now: time.Now} }

func (p *MockProvider) SystemInfo(context.Context) (models.SystemInfo, error) {
	return models.SystemInfo{Hostname: "codenet-router", OS: "CODENET Router OS", Version: "0.1.0-dev", Kernel: "6.12.0-codenet-amd64", Uptime: 9*86400 + 7*3600 + 42*60}, nil
}

func (p *MockProvider) ResourceUsage(context.Context) (models.ResourceUsage, error) {
	return models.ResourceUsage{CPUPercent: 18.4, MemoryPercent: 41.7, MemoryUsed: 895 * 1024 * 1024, MemoryTotal: 2 * 1024 * 1024 * 1024}, nil
}

func (p *MockProvider) Services(context.Context) ([]models.ServiceStatus, error) {
	return []models.ServiceStatus{
		{Name: "codenet-pppoe.service", DisplayName: "Accel-PPP", Status: "active", Description: "PPPoE access concentrator"},
		{Name: "nftables.service", DisplayName: "nftables", Status: "active", Description: "Linux packet filtering"},
		{Name: "systemd-networkd.service", DisplayName: "Network", Status: "active", Description: "Network interface management"},
	}, nil
}

func mockInterfaces() []models.Interface {
	return []models.Interface{
		{Name: "enp0s3", Role: "WAN", Type: "ethernet", MAC: "08:00:27:3a:91:20", IPv4: []string{"10.0.2.15/24"}, Status: "up", MTU: 1500, RXBytes: 2849231024, TXBytes: 847122908, RXPackets: 3981204, TXPackets: 2403811},
		{Name: "enp0s8", Role: "LAN", Type: "ethernet", MAC: "08:00:27:62:f1:0b", IPv4: []string{"10.50.0.1/24"}, Status: "up", MTU: 1500, RXBytes: 1923849902, TXBytes: 2601932944, RXPackets: 2841902, TXPackets: 3702910},
		{Name: "ppp0", Role: "PPPoE", Type: "ppp", MAC: "", IPv4: []string{"10.60.0.1 peer 10.60.0.10"}, Status: "up", MTU: 1492, RXBytes: 580238104, TXBytes: 1294839204, RXPackets: 810292, TXPackets: 1210914},
		{Name: "ppp1", Role: "PPPoE", Type: "ppp", MAC: "", IPv4: []string{"10.60.0.1 peer 10.60.0.11"}, Status: "up", MTU: 1492, RXBytes: 297412805, TXBytes: 792017410, RXPackets: 410201, TXPackets: 721489},
	}
}

func (p *MockProvider) Interfaces(context.Context) ([]models.Interface, error) {
	return mockInterfaces(), nil
}

func (p *MockProvider) Interface(_ context.Context, name string) (models.Interface, error) {
	for _, item := range mockInterfaces() {
		if item.Name == name {
			return item, nil
		}
	}
	return models.Interface{}, NotFoundError{Resource: "interface " + name}
}

func (p *MockProvider) PPPoEStatus(context.Context) (models.PPPoEStatus, error) {
	return models.PPPoEStatus{ServiceName: "CODENET-PPPOE", AccessConcentrator: "CODENET", Status: "active", ActiveSessions: 2, Pool: "10.60.0.10-10.60.0.50"}, nil
}

func (p *MockProvider) PPPoESessions(context.Context) ([]models.PPPoESession, error) {
	now := p.now()
	return []models.PPPoESession{
		{Username: "pelanggan-001", IP: "10.60.0.10", Interface: "ppp0", ConnectedAt: now.Add(-4*time.Hour - 18*time.Minute), UptimeSeconds: 15480, RXBytes: 580238104, TXBytes: 1294839204, Status: "online"},
		{Username: "pelanggan-002", IP: "10.60.0.11", Interface: "ppp1", ConnectedAt: now.Add(-1*time.Hour - 52*time.Minute), UptimeSeconds: 6720, RXBytes: 297412805, TXBytes: 792017410, Status: "online"},
	}, nil
}

func (p *MockProvider) BandwidthProfiles(context.Context) ([]models.BandwidthProfile, error) {
	return []models.BandwidthProfile{{Name: "residential-10m", DownloadBPS: 10_000_000, UploadBPS: 5_000_000, Description: "Profil pelanggan standar"}}, nil
}

func (p *MockProvider) BandwidthStatistics(context.Context) ([]models.BandwidthStatistic, error) {
	return []models.BandwidthStatistic{
		{Username: "pelanggan-001", IP: "10.60.0.10", Interface: "ppp0", DownloadLimit: 10_000_000, UploadLimit: 5_000_000, DownloadUsage: 6_820_000, UploadUsage: 1_940_000, LimiterStatus: "active"},
		{Username: "pelanggan-002", IP: "10.60.0.11", Interface: "ppp1", DownloadLimit: 10_000_000, UploadLimit: 5_000_000, DownloadUsage: 3_410_000, UploadUsage: 810_000, LimiterStatus: "active"},
	}, nil
}

func (p *MockProvider) TrafficHistory(context.Context) ([]models.TrafficPoint, error) {
	now := p.now().Truncate(time.Minute)
	rx := []uint64{3_200_000, 4_100_000, 3_700_000, 5_800_000, 6_300_000, 5_200_000, 7_100_000, 6_700_000, 8_100_000, 7_400_000, 6_800_000, 7_900_000}
	tx := []uint64{1_100_000, 1_500_000, 1_300_000, 2_100_000, 2_400_000, 1_900_000, 2_700_000, 2_300_000, 3_100_000, 2_600_000, 2_200_000, 2_800_000}
	points := make([]models.TrafficPoint, len(rx))
	for i := range rx {
		points[i] = models.TrafficPoint{Timestamp: now.Add(time.Duration(i-len(rx)+1) * 5 * time.Minute), RXBPS: rx[i], TXBPS: tx[i]}
	}
	return points, nil
}
