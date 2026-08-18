package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	expensedomain "github.com/nivra/splitwise-ai/backend/internal/domain/expense"
	groupdomain "github.com/nivra/splitwise-ai/backend/internal/domain/group"
	"github.com/nivra/splitwise-ai/backend/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrExpenseNotFound     = errors.New("expense not found")
	ErrInvalidSplit        = errors.New("split amounts do not add up correctly")
	ErrParticipantNotMember = errors.New("all participants must be active members of the group")
)

type ExpenseParticipantInput struct {
	UserID        uuid.UUID
	AmountMinor   *int64 // for "unequal"
	PercentageBps *int   // for "percentage"
	Shares        *int   // for "shares"
}

type CreateExpenseInput struct {
	GroupID      uuid.UUID
	PaidByUserID uuid.UUID
	CategorySlug string
	Title        string
	Description  *string
	AmountMinor  int64
	Currency     string
	SplitMethod  string
	ExpenseDate  time.Time
	Notes        *string
	Participants []ExpenseParticipantInput
}

type ExpenseService struct {
	groups      repository.GroupRepository
	memberships repository.GroupMembershipRepository
	categories  repository.ExpenseCategoryRepository
	expenses    repository.ExpenseRepository
	shares      repository.ExpenseShareRepository
}

func NewExpenseService(
	groups repository.GroupRepository,
	memberships repository.GroupMembershipRepository,
	categories repository.ExpenseCategoryRepository,
	expenses repository.ExpenseRepository,
	shares repository.ExpenseShareRepository,
) *ExpenseService {
	return &ExpenseService{groups: groups, memberships: memberships, categories: categories, expenses: expenses, shares: shares}
}

func (s *ExpenseService) ListCategories(ctx context.Context) ([]expensedomain.Category, error) {
	return s.categories.List(ctx)
}

func (s *ExpenseService) CreateExpense(ctx context.Context, requesterID uuid.UUID, input CreateExpenseInput) (*expensedomain.Expense, error) {
	group, err := s.groups.FindByID(ctx, input.GroupID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}

	if _, err := s.memberships.FindActive(ctx, input.GroupID, requesterID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotAMember
		}
		return nil, err
	}

	if _, err := s.memberships.FindActive(ctx, input.GroupID, input.PaidByUserID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrParticipantNotMember
		}
		return nil, err
	}
	for _, p := range input.Participants {
		if _, err := s.memberships.FindActive(ctx, input.GroupID, p.UserID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrParticipantNotMember
			}
			return nil, err
		}
	}

	shareAmounts, err := calculateShares(input.SplitMethod, input.AmountMinor, input.Participants)
	if err != nil {
		return nil, err
	}

	var categoryID *uuid.UUID
	if input.CategorySlug != "" {
		cat, err := s.categories.FindBySlug(ctx, input.CategorySlug)
		if err == nil {
			categoryID = &cat.ID
		} else if !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
	}

	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = group.DefaultCurrency
	}

	newExpense := &expensedomain.Expense{
		GroupID:         input.GroupID,
		CreatedByUserID: requesterID,
		PaidByUserID:    input.PaidByUserID,
		CategoryID:      categoryID,
		Title:           strings.TrimSpace(input.Title),
		Description:     input.Description,
		AmountMinor:     input.AmountMinor,
		Currency:        currency,
		SplitMethod:     input.SplitMethod,
		ExpenseDate:     input.ExpenseDate,
		Notes:           input.Notes,
		Status:          expensedomain.StatusActive,
	}
	if err := s.expenses.Create(ctx, newExpense); err != nil {
		return nil, err
	}

	shareRows := make([]expensedomain.Share, 0, len(input.Participants))
	for i, p := range input.Participants {
		row := expensedomain.Share{
			ExpenseID:        newExpense.ID,
			UserID:           p.UserID,
			ShareAmountMinor: shareAmounts[i],
		}
		if input.SplitMethod == expensedomain.SplitPercentage {
			row.SharePercentageBps = p.PercentageBps
		}
		if input.SplitMethod == expensedomain.SplitShares {
			row.ShareCount = p.Shares
		}
		shareRows = append(shareRows, row)
	}
	if err := s.shares.CreateBatch(ctx, shareRows); err != nil {
		return nil, err
	}

	return newExpense, nil
}

func (s *ExpenseService) ListExpenses(ctx context.Context, requesterID, groupID uuid.UUID) ([]expensedomain.Expense, error) {
	if _, err := s.memberships.FindActive(ctx, groupID, requesterID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotAMember
		}
		return nil, err
	}
	return s.expenses.ListByGroup(ctx, groupID)
}

func (s *ExpenseService) DeleteExpense(ctx context.Context, requesterID, expenseID uuid.UUID) error {
	found, err := s.expenses.FindByID(ctx, expenseID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrExpenseNotFound
	}
	if err != nil {
		return err
	}

	membership, err := s.memberships.FindActive(ctx, found.GroupID, requesterID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotAMember
	}
	if err != nil {
		return err
	}

	canDelete := found.CreatedByUserID == requesterID ||
		found.PaidByUserID == requesterID ||
		membership.Role == groupdomain.MembershipRoleOwner ||
		membership.Role == groupdomain.MembershipRoleAdmin
	if !canDelete {
		return ErrForbidden
	}

	return s.expenses.SoftDelete(ctx, expenseID, time.Now().UTC())
}

type UserBalance struct {
	UserID   uuid.UUID
	NetMinor int64 // positive = owed to them, negative = they owe the group
}

// GetBalances computes each member's net position from expenses paid vs.
// share owed. It does not yet net out recorded settlements — that lands with
// the settlement-recording feature.
func (s *ExpenseService) GetBalances(ctx context.Context, requesterID, groupID uuid.UUID) ([]UserBalance, error) {
	if _, err := s.memberships.FindActive(ctx, groupID, requesterID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotAMember
		}
		return nil, err
	}

	paid, err := s.expenses.SumPaidByUser(ctx, groupID)
	if err != nil {
		return nil, err
	}
	owed, err := s.shares.SumShareByUser(ctx, groupID)
	if err != nil {
		return nil, err
	}

	userIDs := map[uuid.UUID]bool{}
	for id := range paid {
		userIDs[id] = true
	}
	for id := range owed {
		userIDs[id] = true
	}

	balances := make([]UserBalance, 0, len(userIDs))
	for id := range userIDs {
		balances = append(balances, UserBalance{UserID: id, NetMinor: paid[id] - owed[id]})
	}
	return balances, nil
}

// calculateShares turns the split method plus per-participant input into an
// exact per-participant amount (in minor units) that always sums to
// totalMinor — proportional methods use the largest-remainder method so
// rounding never lets the total drift by a paisa.
func calculateShares(splitMethod string, totalMinor int64, participants []ExpenseParticipantInput) ([]int64, error) {
	if len(participants) == 0 {
		return nil, ErrInvalidSplit
	}

	switch splitMethod {
	case expensedomain.SplitEqual:
		weights := make([]int64, len(participants))
		for i := range weights {
			weights[i] = 1
		}
		return allocateProportional(totalMinor, weights), nil

	case expensedomain.SplitShares:
		weights := make([]int64, len(participants))
		for i, p := range participants {
			if p.Shares == nil || *p.Shares <= 0 {
				return nil, ErrInvalidSplit
			}
			weights[i] = int64(*p.Shares)
		}
		return allocateProportional(totalMinor, weights), nil

	case expensedomain.SplitPercentage:
		weights := make([]int64, len(participants))
		sumBps := 0
		for i, p := range participants {
			if p.PercentageBps == nil || *p.PercentageBps < 0 {
				return nil, ErrInvalidSplit
			}
			weights[i] = int64(*p.PercentageBps)
			sumBps += *p.PercentageBps
		}
		if sumBps != 10000 {
			return nil, ErrInvalidSplit
		}
		return allocateProportional(totalMinor, weights), nil

	case expensedomain.SplitUnequal:
		amounts := make([]int64, len(participants))
		var sum int64
		for i, p := range participants {
			if p.AmountMinor == nil || *p.AmountMinor <= 0 {
				return nil, ErrInvalidSplit
			}
			amounts[i] = *p.AmountMinor
			sum += *p.AmountMinor
		}
		if sum != totalMinor {
			return nil, ErrInvalidSplit
		}
		return amounts, nil

	default:
		return nil, ErrInvalidSplit
	}
}

func allocateProportional(totalMinor int64, weights []int64) []int64 {
	var totalWeight int64
	for _, w := range weights {
		totalWeight += w
	}
	if totalWeight <= 0 {
		return make([]int64, len(weights))
	}

	allocations := make([]int64, len(weights))
	remainders := make([]float64, len(weights))
	var allocatedSum int64
	for i, w := range weights {
		exact := float64(totalMinor) * float64(w) / float64(totalWeight)
		floor := int64(exact)
		allocations[i] = floor
		remainders[i] = exact - float64(floor)
		allocatedSum += floor
	}

	remaining := totalMinor - allocatedSum
	order := make([]int, len(weights))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool { return remainders[order[a]] > remainders[order[b]] })
	for i := int64(0); i < remaining; i++ {
		allocations[order[i]]++
	}
	return allocations
}
