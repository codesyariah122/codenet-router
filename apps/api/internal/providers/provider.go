package providers

import (
	"context"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
)

// RouterProvider adalah batas read-only terhadap sistem operasi router.
// Handler HTTP tidak boleh mengakses /proc, /sys, systemd, accel-ppp, atau tc secara langsung.
type RouterProvider interface {
	SystemInfo(context.Context) (models.SystemInfo, error)
	ResourceUsage(context.Context) (models.ResourceUsage, error)
	Services(context.Context) ([]models.ServiceStatus, error)
	Interfaces(context.Context) ([]models.Interface, error)
	Interface(context.Context, string) (models.Interface, error)
	IPAddresses(context.Context) ([]models.IPAddress, error)
	Routes(context.Context) ([]models.Route, error)
	PPPoEStatus(context.Context) (models.PPPoEStatus, error)
	PPPoESessions(context.Context) ([]models.PPPoESession, error)
	BandwidthProfiles(context.Context) ([]models.BandwidthProfile, error)
	BandwidthStatistics(context.Context) ([]models.BandwidthStatistic, error)
	TrafficHistory(context.Context) ([]models.TrafficPoint, error)
}

type NotFoundError struct{ Resource string }

func (e NotFoundError) Error() string { return e.Resource + " tidak ditemukan" }
