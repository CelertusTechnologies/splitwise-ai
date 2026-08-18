package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	expensedomain "github.com/nivra/splitwise-ai/backend/internal/domain/expense"
	"github.com/nivra/splitwise-ai/backend/internal/service"
	"github.com/nivra/splitwise-ai/backend/internal/transport/http/response"
)

type ExpenseHandler struct {
	expenses *service.ExpenseService
}

func NewExpenseHandler(expenses *service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{expenses: expenses}
}

type expenseParticipantRequest struct {
	UserID     string   `json:"user_id" binding:"required,uuid"`
	Amount     *float64 `json:"amount" binding:"omitempty,gt=0"`
	Percentage *float64 `json:"percentage" binding:"omitempty,gte=0"`
	Shares     *int     `json:"shares" binding:"omitempty,gt=0"`
}

type createExpenseRequest struct {
	Title        string                      `json:"title" binding:"required,min=1,max=200"`
	Description  *string                     `json:"description" binding:"omitempty,max=500"`
	Amount       float64                     `json:"amount" binding:"required,gt=0"`
	Currency     string                      `json:"currency" binding:"omitempty,len=3"`
	CategorySlug string                      `json:"category_slug"`
	PaidByUserID string                      `json:"paid_by_user_id" binding:"required,uuid"`
	SplitMethod  string                      `json:"split_method" binding:"required,oneof=equal unequal percentage shares"`
	ExpenseDate  string                      `json:"expense_date" binding:"required"`
	Notes        *string                     `json:"notes" binding:"omitempty,max=1000"`
	Participants []expenseParticipantRequest `json:"participants" binding:"required,min=1,dive"`
}

type expenseResponse struct {
	ID           string    `json:"id"`
	GroupID      string    `json:"group_id"`
	Title        string    `json:"title"`
	Description  *string   `json:"description"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	SplitMethod  string    `json:"split_method"`
	PaidByUserID string    `json:"paid_by_user_id"`
	ExpenseDate  string    `json:"expense_date"`
	CreatedAt    time.Time `json:"created_at"`
}

func toExpenseResponse(e expensedomain.Expense) expenseResponse {
	return expenseResponse{
		ID:           e.ID.String(),
		GroupID:      e.GroupID.String(),
		Title:        e.Title,
		Description:  e.Description,
		Amount:       minorToAmount(e.AmountMinor),
		Currency:     e.Currency,
		SplitMethod:  e.SplitMethod,
		PaidByUserID: e.PaidByUserID.String(),
		ExpenseDate:  e.ExpenseDate.Format("2006-01-02"),
		CreatedAt:    e.CreatedAt,
	}
}

func minorToAmount(minor int64) float64 {
	return float64(minor) / 100
}

func amountToMinor(amount float64) int64 {
	return int64(amount*100 + 0.5) // round to the nearest paisa
}

func (h *ExpenseHandler) Create(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", "invalid group id")
		return
	}

	var req createExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	paidBy, err := uuid.Parse(req.PaidByUserID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", "invalid paid_by_user_id")
		return
	}

	expenseDate, err := time.Parse("2006-01-02", req.ExpenseDate)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", "expense_date must be YYYY-MM-DD")
		return
	}

	participants := make([]service.ExpenseParticipantInput, 0, len(req.Participants))
	for _, p := range req.Participants {
		participantUserID, err := uuid.Parse(p.UserID)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "validation_failed", "invalid participant user_id")
			return
		}

		participant := service.ExpenseParticipantInput{UserID: participantUserID}
		if p.Amount != nil {
			amountMinor := amountToMinor(*p.Amount)
			participant.AmountMinor = &amountMinor
		}
		if p.Percentage != nil {
			bps := int(*p.Percentage*100 + 0.5)
			participant.PercentageBps = &bps
		}
		if p.Shares != nil {
			participant.Shares = p.Shares
		}
		participants = append(participants, participant)
	}

	created, err := h.expenses.CreateExpense(c.Request.Context(), userID, service.CreateExpenseInput{
		GroupID:      groupID,
		PaidByUserID: paidBy,
		CategorySlug: req.CategorySlug,
		Title:        req.Title,
		Description:  req.Description,
		AmountMinor:  amountToMinor(req.Amount),
		Currency:     req.Currency,
		SplitMethod:  req.SplitMethod,
		ExpenseDate:  expenseDate,
		Notes:        req.Notes,
		Participants: participants,
	})
	if writeExpenseServiceError(c, err) {
		return
	}

	response.Created(c, gin.H{"expense": toExpenseResponse(*created)})
}

func (h *ExpenseHandler) List(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", "invalid group id")
		return
	}

	found, err := h.expenses.ListExpenses(c.Request.Context(), userID, groupID)
	if writeExpenseServiceError(c, err) {
		return
	}

	items := make([]expenseResponse, 0, len(found))
	for _, e := range found {
		items = append(items, toExpenseResponse(e))
	}
	response.OK(c, gin.H{"expenses": items})
}

func (h *ExpenseHandler) Delete(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", "invalid expense id")
		return
	}

	if err := h.expenses.DeleteExpense(c.Request.Context(), userID, expenseID); writeExpenseServiceError(c, err) {
		return
	}

	response.NoContent(c)
}

type balanceResponse struct {
	UserID string  `json:"user_id"`
	Net    float64 `json:"net"`
}

func (h *ExpenseHandler) Balances(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", "invalid group id")
		return
	}

	balances, err := h.expenses.GetBalances(c.Request.Context(), userID, groupID)
	if writeExpenseServiceError(c, err) {
		return
	}

	items := make([]balanceResponse, 0, len(balances))
	for _, b := range balances {
		items = append(items, balanceResponse{UserID: b.UserID.String(), Net: minorToAmount(b.NetMinor)})
	}
	response.OK(c, gin.H{"balances": items})
}

type categoryResponse struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

func (h *ExpenseHandler) ListCategories(c *gin.Context) {
	found, err := h.expenses.ListCategories(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", "Could not load categories.")
		return
	}

	items := make([]categoryResponse, 0, len(found))
	for _, cat := range found {
		items = append(items, categoryResponse{Slug: cat.Slug, Name: cat.Name, Icon: cat.Icon})
	}
	response.OK(c, gin.H{"categories": items})
}

func writeExpenseServiceError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, service.ErrExpenseNotFound):
		response.Error(c, http.StatusNotFound, "expense_not_found", "Expense not found.")
	case errors.Is(err, service.ErrGroupNotFound), errors.Is(err, service.ErrNotAMember):
		response.Error(c, http.StatusNotFound, "group_not_found", "Group not found.")
	case errors.Is(err, service.ErrParticipantNotMember):
		response.Error(c, http.StatusBadRequest, "invalid_participants", "All participants must be active members of the group.")
	case errors.Is(err, service.ErrInvalidSplit):
		response.Error(c, http.StatusBadRequest, "invalid_split", "The split amounts don't add up to the total.")
	case errors.Is(err, service.ErrForbidden):
		response.Error(c, http.StatusForbidden, "forbidden", "You don't have permission to do that.")
	default:
		response.Error(c, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
	return true
}
