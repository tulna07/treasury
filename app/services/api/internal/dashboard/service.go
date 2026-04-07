package dashboard

import (
	"context"
	"fmt"
)

// DashboardResponse is the aggregated dashboard payload.
type DashboardResponse struct {
	Summary            *SummaryToday       `json:"summary"`
	DailyVolume        []DailyVolume       `json:"daily_volume"`
	ModuleDistribution *ModuleDistribution `json:"module_distribution"`
	StatusDaily        []StatusDaily       `json:"status_daily"`
	RecentTransactions []RecentTransaction `json:"recent_transactions"`
}

// Service orchestrates dashboard data retrieval.
type Service struct {
	repo *Repository
}

// NewService creates a new dashboard service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetDashboard fetches all dashboard sections sequentially.
func (s *Service) GetDashboard(ctx context.Context) (*DashboardResponse, error) {
	summary, err := s.repo.GetSummaryToday(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard summary: %w", err)
	}

	dailyVolume, err := s.repo.GetDailyVolume(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard daily volume: %w", err)
	}

	moduleDist, err := s.repo.GetModuleDistribution(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard module distribution: %w", err)
	}

	statusDaily, err := s.repo.GetStatusDaily(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard status daily: %w", err)
	}

	recentTx, err := s.repo.GetRecentTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard recent transactions: %w", err)
	}

	return &DashboardResponse{
		Summary:            summary,
		DailyVolume:        dailyVolume,
		ModuleDistribution: moduleDist,
		StatusDaily:        statusDaily,
		RecentTransactions: recentTx,
	}, nil
}
