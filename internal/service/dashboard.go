package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
	"github.com/yourname/tabikake/internal/store"
)

// DashboardService computes aggregated trip statistics.
type DashboardService struct {
	db     *store.DB
	notion *notion.Client
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(db *store.DB, notionClient *notion.Client) *DashboardService {
	return &DashboardService{db: db, notion: notionClient}
}

// GetDashboard returns spending summary for a trip.
func (s *DashboardService) GetDashboard(ctx context.Context, tripID string) (*model.DashboardResponse, error) {
	trip, err := s.db.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	records, err := s.notion.ListRecords(ctx, trip.NotionDbID)
	if err != nil {
		return nil, err
	}

	// Collect user IDs.
	idSet := make(map[string]struct{})
	for _, r := range records {
		if r.PaidByUserID != "" {
			idSet[r.PaidByUserID] = struct{}{}
		}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	users, _ := s.db.GetUsersByIDs(ctx, ids)

	// Enrich records with user info.
	for i := range records {
		if u, ok := users[records[i].PaidByUserID]; ok {
			u := u
			records[i].PaidByUser = &u
		}
	}

	return buildDashboard(records, users), nil
}

func buildDashboard(records []model.Record, users map[string]model.User) *model.DashboardResponse {
	var totalJPY int64
	paidMap := make(map[string]int64)
	catMap := make(map[string]int64)
	dateMap := make(map[string]int64)

	for _, r := range records {
		amt := int64(r.AmountJPY)
		totalJPY += amt
		paidMap[r.PaidByUserID] += amt
		catMap[r.Category] += amt
		dateMap[r.Date] += amt
	}

	// ByMember
	byMember := make([]model.ByMember, 0, len(paidMap))
	for uid, paid := range paidMap {
		pct := 0.0
		if totalJPY > 0 {
			pct = float64(paid) / float64(totalJPY) * 100
		}
		bm := model.ByMember{PaidJPY: paid, Percentage: pct}
		if u, ok := users[uid]; ok {
			bm.User = u
		} else {
			bm.User = model.User{ID: uid, Name: uid}
		}
		byMember = append(byMember, bm)
	}
	sort.Slice(byMember, func(i, j int) bool { return byMember[i].PaidJPY > byMember[j].PaidJPY })

	// ByCategory
	byCategory := make([]model.ByCategory, 0, len(catMap))
	for cat, amt := range catMap {
		byCategory = append(byCategory, model.ByCategory{Category: cat, AmountJPY: amt})
	}
	sort.Slice(byCategory, func(i, j int) bool { return byCategory[i].AmountJPY > byCategory[j].AmountJPY })

	// ByDate
	byDate := make([]model.ByDate, 0, len(dateMap))
	for date, amt := range dateMap {
		byDate = append(byDate, model.ByDate{Date: date, AmountJPY: amt})
	}
	sort.Slice(byDate, func(i, j int) bool { return byDate[i].Date < byDate[j].Date })

	return &model.DashboardResponse{
		TotalJPY:   totalJPY,
		ByMember:   byMember,
		ByCategory: byCategory,
		ByDate:     byDate,
		Records:    records,
	}
}
