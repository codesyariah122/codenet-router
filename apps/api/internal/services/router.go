package services

import (
	"context"

	"github.com/codesyariah/codenet-router/apps/api/internal/models"
	"github.com/codesyariah/codenet-router/apps/api/internal/providers"
)

type RouterService struct{ provider providers.RouterProvider }

func NewRouter(provider providers.RouterProvider) *RouterService {
	return &RouterService{provider: provider}
}

func (s *RouterService) Dashboard(ctx context.Context) (models.Dashboard, error) {
	info, err := s.provider.SystemInfo(ctx)
	if err != nil {
		return models.Dashboard{}, err
	}
	resources, err := s.provider.ResourceUsage(ctx)
	if err != nil {
		return models.Dashboard{}, err
	}
	interfaces, err := s.provider.Interfaces(ctx)
	if err != nil {
		return models.Dashboard{}, err
	}
	pppoe, err := s.provider.PPPoEStatus(ctx)
	if err != nil {
		return models.Dashboard{}, err
	}
	traffic, err := s.provider.TrafficHistory(ctx)
	if err != nil {
		return models.Dashboard{}, err
	}
	return models.Dashboard{System: info, Resources: resources, Interfaces: interfaces, PPPoE: pppoe, TrafficHistory: traffic}, nil
}

func (s *RouterService) Provider() providers.RouterProvider { return s.provider }
