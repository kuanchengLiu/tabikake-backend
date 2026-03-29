package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
	"github.com/yourname/tabikake/internal/store"
)

// SettlementService calculates and exports trip settlements.
type SettlementService struct {
	db     *store.DB
	notion *notion.Client
}

// NewSettlementService creates a new SettlementService.
func NewSettlementService(db *store.DB, notionClient *notion.Client) *SettlementService {
	return &SettlementService{db: db, notion: notionClient}
}

// Calculate returns settlement results without writing to Notion.
func (s *SettlementService) Calculate(ctx context.Context, tripID string) (*model.SettlementResult, error) {
	trip, err := s.db.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	records, err := s.notion.ListRecords(ctx, trip.NotionDbID)
	if err != nil {
		return nil, err
	}

	members, err := s.db.ListMembers(ctx, tripID)
	if err != nil {
		return nil, err
	}
	memberCount := len(members)
	if memberCount == 0 {
		return &model.SettlementResult{}, nil
	}

	// Collect all involved user IDs.
	idSet := make(map[string]struct{})
	for _, m := range members {
		idSet[m.UserID] = struct{}{}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	users, _ := s.db.GetUsersByIDs(ctx, ids)

	result, totalJPY := calculateSettlements(records, members, users)
	return &model.SettlementResult{TotalJPY: totalJPY, Settlements: result}, nil
}

// Export calculates settlement and creates a Notion summary page.
func (s *SettlementService) Export(ctx context.Context, tripID string) (*model.ExportSettlementResponse, error) {
	trip, err := s.db.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	records, err := s.notion.ListRecords(ctx, trip.NotionDbID)
	if err != nil {
		return nil, err
	}

	members, err := s.db.ListMembers(ctx, tripID)
	if err != nil {
		return nil, err
	}

	idSet := make(map[string]struct{})
	for _, m := range members {
		idSet[m.UserID] = struct{}{}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	users, _ := s.db.GetUsersByIDs(ctx, ids)

	settlements, totalJPY := calculateSettlements(records, members, users)

	// Build per-user record lists.
	recordsByUser := make(map[string][]model.Record)
	for _, r := range records {
		recordsByUser[r.PaidByUserID] = append(recordsByUser[r.PaidByUserID], r)
	}

	byUser := make([]notion.UserRecords, 0, len(members))
	for _, m := range members {
		u, ok := users[m.UserID]
		if !ok {
			continue
		}
		recs := recordsByUser[m.UserID]
		var total int64
		for _, r := range recs {
			total += int64(r.AmountJPY)
		}
		byUser = append(byUser, notion.UserRecords{User: u, Records: recs, TotalJPY: total})
	}

	pageURL, err := s.notion.CreateSettlementPage(ctx, trip.NotionPageID, trip.Name, notion.SettlementExportData{
		Settlements: settlements,
		ByUser:      byUser,
		TotalJPY:    totalJPY,
	})
	if err != nil {
		return nil, fmt.Errorf("create settlement page: %w", err)
	}

	return &model.ExportSettlementResponse{NotionPageURL: pageURL}, nil
}

func calculateSettlements(records []model.Record, members []model.Member, users map[string]model.User) ([]model.SettlementItem, int64) {
	// paid[userID] = total actually paid
	// owe[userID]  = total should pay
	paid := make(map[string]int64)
	owe := make(map[string]int64)
	var totalJPY int64

	for _, r := range records {
		amt := int64(r.AmountJPY)
		totalJPY += amt
		paid[r.PaidByUserID] += amt

		// AA制: split_with empty → split among all members
		// 自選: split_with指定成員
		splitAmong := r.SplitWith
		if len(splitAmong) == 0 {
			for _, m := range members {
				splitAmong = append(splitAmong, m.UserID)
			}
		}
		if len(splitAmong) > 0 {
			share := amt / int64(len(splitAmong))
			remainder := amt - share*int64(len(splitAmong))
			for i, uid := range splitAmong {
				s := share
				if i == 0 {
					s += remainder
				}
				owe[uid] += s
			}
		}
	}

	// net balance = paid - owe (positive = others owe you)
	balances := make([]model.UserBalance, 0, len(users))
	for _, m := range members {
		u, ok := users[m.UserID]
		if !ok {
			u = model.User{ID: m.UserID, Name: m.UserID}
		}
		balances = append(balances, model.UserBalance{
			User:    u,
			Balance: paid[m.UserID] - owe[m.UserID],
		})
	}

	return greedySettle(balances), totalJPY
}

func greedySettle(balances []model.UserBalance) []model.SettlementItem {
	type entry struct {
		user   model.User
		amount int64
	}

	var creditors, debtors []entry
	for _, b := range balances {
		if b.Balance > 0 {
			creditors = append(creditors, entry{b.User, b.Balance})
		} else if b.Balance < 0 {
			debtors = append(debtors, entry{b.User, -b.Balance})
		}
	}

	sort.Slice(creditors, func(i, j int) bool { return creditors[i].amount > creditors[j].amount })
	sort.Slice(debtors, func(i, j int) bool { return debtors[i].amount > debtors[j].amount })

	var result []model.SettlementItem
	i, j := 0, 0
	for i < len(creditors) && j < len(debtors) {
		amt := creditors[i].amount
		if debtors[j].amount < amt {
			amt = debtors[j].amount
		}
		result = append(result, model.SettlementItem{
			From:      debtors[j].user,
			To:        creditors[i].user,
			AmountJPY: amt,
		})
		creditors[i].amount -= amt
		debtors[j].amount -= amt
		if creditors[i].amount == 0 {
			i++
		}
		if debtors[j].amount == 0 {
			j++
		}
	}
	return result
}
