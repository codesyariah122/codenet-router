//go:build !linux

package providers

import (
	"context"
	"errors"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
)

// NewLinux tersedia agar pemilihan provider gagal secara eksplisit di non-Linux.
// Mode pengembangan default selalu menggunakan mock provider.
func NewLinux() RouterProvider { return &unsupportedLinuxProvider{} }

type unsupportedLinuxProvider struct{}

var errLinuxUnsupported = errors.New("linux provider hanya tersedia pada sistem operasi Linux")

func (*unsupportedLinuxProvider) SystemInfo(context.Context) (models.SystemInfo, error) {
	return models.SystemInfo{}, errLinuxUnsupported
}
func (*unsupportedLinuxProvider) ResourceUsage(context.Context) (models.ResourceUsage, error) {
	return models.ResourceUsage{}, errLinuxUnsupported
}
func (*unsupportedLinuxProvider) Services(context.Context) ([]models.ServiceStatus, error) {
	return nil, errLinuxUnsupported
}
func (*unsupportedLinuxProvider) Interfaces(context.Context) ([]models.Interface, error) {
	return nil, errLinuxUnsupported
}
func (*unsupportedLinuxProvider) Interface(context.Context, string) (models.Interface, error) {
	return models.Interface{}, errLinuxUnsupported
}
func (*unsupportedLinuxProvider) PPPoEStatus(context.Context) (models.PPPoEStatus, error) {
	return models.PPPoEStatus{}, errLinuxUnsupported
}
func (*unsupportedLinuxProvider) PPPoESessions(context.Context) ([]models.PPPoESession, error) {
	return nil, errLinuxUnsupported
}
func (*unsupportedLinuxProvider) BandwidthProfiles(context.Context) ([]models.BandwidthProfile, error) {
	return nil, errLinuxUnsupported
}
func (*unsupportedLinuxProvider) BandwidthStatistics(context.Context) ([]models.BandwidthStatistic, error) {
	return nil, errLinuxUnsupported
}
func (*unsupportedLinuxProvider) TrafficHistory(context.Context) ([]models.TrafficPoint, error) {
	return nil, errLinuxUnsupported
}
