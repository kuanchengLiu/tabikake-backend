package service

import (
	"context"
	"fmt"

	"github.com/yourname/tabikake/internal/db"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
)

// SplitService exports settlement results as a Notion page.
type SplitService struct {
	db        *db.DB
	notion    *notion.Client
	dashboard *DashboardService
}

// NewSplitService creates a new SplitService.
func NewSplitService(database *db.DB, notionClient *notion.Client, dashboard *DashboardService) *SplitService {
	return &SplitService{db: database, notion: notionClient, dashboard: dashboard}
}

// ExportSettlement calculates the settlement for a trip and creates a summary
// Notion page under the trip's parent page. Returns the page URL.
func (s *SplitService) ExportSettlement(ctx context.Context, tripID string) (*model.ExportSettlementResponse, error) {
	trip, err := s.db.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	dash, err := s.dashboard.GetDashboard(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get dashboard: %w", err)
	}

	pageURL, err := s.notion.CreateSettlementPage(ctx, trip.NotionPageID, trip.Name, notion.SettlementData{
		Settlements: dash.Settlements,
		Balances:    dash.MemberBalances,
		TotalJPY:    dash.TotalJPY,
	})
	if err != nil {
		return nil, fmt.Errorf("create settlement page: %w", err)
	}

	return &model.ExportSettlementResponse{NotionPageURL: pageURL}, nil
}
