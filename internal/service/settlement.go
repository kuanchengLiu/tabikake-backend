package service

import (
	"context"
	"fmt"
	"sort"

	appdb "github.com/yourname/tabikake/internal/db"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
)

// SettlementService calculates member-based trip settlements.
type SettlementService struct {
	db     *appdb.DB
	notion *notion.Client
}

// NewSettlementService creates a new SettlementService.
func NewSettlementService(database *appdb.DB, notionClient *notion.Client) *SettlementService {
	return &SettlementService{db: database, notion: notionClient}
}

// GetSettlement computes per-member balances and the minimum transfers needed.
func (s *SettlementService) GetSettlement(ctx context.Context, tripID string) (*model.SettlementResult, error) {
	trip, err := s.db.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	members, err := s.db.ListMembers(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	if len(members) == 0 {
		return &model.SettlementResult{}, nil
	}

	records, err := s.notion.ListRecords(ctx, trip.NotionDbID)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}

	// Index members by ID for quick lookup.
	memberMap := make(map[string]model.Member, len(members))
	for _, m := range members {
		memberMap[m.ID] = m
	}

	// paid[memberID] = total amount this member actually paid
	// owe[memberID]  = total amount this member should pay (based on splits)
	paid := make(map[string]int64, len(members))
	owe := make(map[string]int64, len(members))

	var totalJPY int64

	for _, r := range records {
		amount := int64(r.AmountJPY)
		totalJPY += amount

		// Credit the payer.
		if r.PaidByMemberID != "" {
			paid[r.PaidByMemberID] += amount
		}

		// Distribute the cost.
		if len(r.SplitWith) == 0 {
			// Not split — only payer bears the cost.
			if r.PaidByMemberID != "" {
				owe[r.PaidByMemberID] += amount
			}
		} else {
			// Split equally among split_with members.
			share := amount / int64(len(r.SplitWith))
			remainder := amount - share*int64(len(r.SplitWith))
			for i, mid := range r.SplitWith {
				s := share
				if i == 0 {
					s += remainder // first member absorbs rounding
				}
				owe[mid] += s
			}
		}
	}

	// Build per-member summaries.
	summaries := make([]model.MemberSummary, 0, len(members))
	balances := make(map[string]int64, len(members))
	for _, m := range members {
		p := paid[m.ID]
		o := owe[m.ID]
		diff := p - o
		balances[m.ID] = diff
		summaries = append(summaries, model.MemberSummary{
			Member:  m,
			PaidJPY: p,
			OweJPY:  o,
			DiffJPY: diff,
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].DiffJPY > summaries[j].DiffJPY
	})

	return &model.SettlementResult{
		TotalJPY:    totalJPY,
		ByMember:    summaries,
		Settlements: memberGreedySettle(balances, memberMap),
	}, nil
}

func memberGreedySettle(balances map[string]int64, memberMap map[string]model.Member) []model.MemberSettlement {
	type entry struct {
		id     string
		amount int64
	}
	var creditors, debtors []entry
	for id, bal := range balances {
		if bal > 0 {
			creditors = append(creditors, entry{id, bal})
		} else if bal < 0 {
			debtors = append(debtors, entry{id, -bal})
		}
	}
	sort.Slice(creditors, func(i, j int) bool { return creditors[i].amount > creditors[j].amount })
	sort.Slice(debtors, func(i, j int) bool { return debtors[i].amount > debtors[j].amount })

	var result []model.MemberSettlement
	i, j := 0, 0
	for i < len(creditors) && j < len(debtors) {
		amount := creditors[i].amount
		if debtors[j].amount < amount {
			amount = debtors[j].amount
		}
		result = append(result, model.MemberSettlement{
			From:      memberMap[debtors[j].id],
			To:        memberMap[creditors[i].id],
			AmountJPY: amount,
		})
		creditors[i].amount -= amount
		debtors[j].amount -= amount
		if creditors[i].amount == 0 {
			i++
		}
		if debtors[j].amount == 0 {
			j++
		}
	}
	return result
}
