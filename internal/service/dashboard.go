package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/yourname/tabikake/internal/db"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
)

// DashboardService computes aggregated trip statistics and settlement.
type DashboardService struct {
	db     *db.DB
	notion *notion.Client
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(database *db.DB, notionClient *notion.Client) *DashboardService {
	return &DashboardService{db: database, notion: notionClient}
}

// GetDashboard returns spending summary and settlement for a trip.
func (s *DashboardService) GetDashboard(ctx context.Context, tripID string) (*model.DashboardResponse, error) {
	trip, err := s.db.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	records, err := s.notion.ListRecords(ctx, trip.NotionDbID)
	if err != nil {
		return nil, err
	}

	return buildDashboard(tripID, records), nil
}

func buildDashboard(tripID string, records []model.Record) *model.DashboardResponse {
	var totalJPY float64
	paidMap := make(map[string]float64)
	nameMap := make(map[string]string)
	categoryMap := make(map[string]float64)

	for _, r := range records {
		totalJPY += r.AmountJPY
		paidMap[r.PaidBy] += r.AmountJPY
		nameMap[r.PaidBy] = r.PaidByName
		categoryMap[r.Category] += r.AmountJPY
	}

	memberCount := float64(len(paidMap))
	if memberCount == 0 {
		memberCount = 1
	}
	perPerson := totalJPY / memberCount

	balances := make([]model.MemberBalance, 0, len(paidMap))
	for userID, paid := range paidMap {
		name := nameMap[userID]
		if name == "" {
			name = userID
		}
		balances = append(balances, model.MemberBalance{
			UserID:    userID,
			UserName:  name,
			TotalPaid: paid,
			ShouldPay: perPerson,
			Balance:   paid - perPerson,
		})
	}
	sort.Slice(balances, func(i, j int) bool {
		return balances[i].Balance > balances[j].Balance
	})

	categories := make([]model.CategorySummary, 0, len(categoryMap))
	for cat, amt := range categoryMap {
		categories = append(categories, model.CategorySummary{Category: cat, AmountJPY: amt})
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].AmountJPY > categories[j].AmountJPY
	})

	return &model.DashboardResponse{
		TripID:          tripID,
		TotalJPY:        totalJPY,
		MemberBalances:  balances,
		CategorySummary: categories,
		Settlements:     greedySettle(balances),
	}
}

func greedySettle(balances []model.MemberBalance) []model.Settlement {
	type entry struct {
		userID   string
		userName string
		amount   float64
	}

	const epsilon = 0.01
	var creditors, debtors []entry

	for _, b := range balances {
		if b.Balance > epsilon {
			creditors = append(creditors, entry{b.UserID, b.UserName, b.Balance})
		} else if b.Balance < -epsilon {
			debtors = append(debtors, entry{b.UserID, b.UserName, -b.Balance})
		}
	}

	var settlements []model.Settlement
	i, j := 0, 0
	for i < len(creditors) && j < len(debtors) {
		transfer := min64(creditors[i].amount, debtors[j].amount)
		settlements = append(settlements, model.Settlement{
			FromUserID:   debtors[j].userID,
			FromUserName: debtors[j].userName,
			ToUserID:     creditors[i].userID,
			ToUserName:   creditors[i].userName,
			Amount:       roundJPY(transfer),
		})
		creditors[i].amount -= transfer
		debtors[j].amount -= transfer
		if creditors[i].amount < epsilon {
			i++
		}
		if debtors[j].amount < epsilon {
			j++
		}
	}
	return settlements
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func roundJPY(v float64) float64 {
	return float64(int64(v + 0.5))
}
