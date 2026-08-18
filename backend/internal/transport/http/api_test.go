package httptransport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nivra/splitwise-ai/backend/internal/config"
	"github.com/nivra/splitwise-ai/backend/internal/platform/database"
	applogger "github.com/nivra/splitwise-ai/backend/internal/platform/logger"
	"github.com/nivra/splitwise-ai/backend/internal/platform/security"
	"github.com/nivra/splitwise-ai/backend/internal/repository"
	"github.com/nivra/splitwise-ai/backend/internal/service"
	httptransport "github.com/nivra/splitwise-ai/backend/internal/transport/http"
	"github.com/nivra/splitwise-ai/backend/internal/transport/http/handlers"
)

// newTestServer boots the exact same wiring as cmd/api/main.go, but against
// an isolated in-memory SQLite database, and returns the base URL of a live
// httptest server. Each call gets its own database, so tests never leak
// state into one another.
func newTestServer(t *testing.T) string {
	t.Helper()

	cfg := config.Config{
		AppName:          "Nivra",
		Env:              "test",
		DBDriver:         config.DriverSQLite,
		DatabaseURL:      ":memory:",
		FrontendURL:      "http://localhost:3000",
		JWTAccessSecret:  "test-access-secret",
		JWTRefreshSecret: "test-refresh-secret",
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  30 * 24 * time.Hour,
	}

	logger, err := applogger.New(cfg.Env)
	if err != nil {
		t.Fatalf("logger init: %v", err)
	}

	db, err := database.Connect(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}

	jwtManager := security.NewJWTManager(cfg)
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	oneTimeTokenRepo := repository.NewOneTimeTokenRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	groupMembershipRepo := repository.NewGroupMembershipRepository(db)
	groupInviteRepo := repository.NewGroupInviteRepository(db)
	phoneOTPRepo := repository.NewPhoneOTPRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	expenseShareRepo := repository.NewExpenseShareRepository(db)
	expenseCategoryRepo := repository.NewExpenseCategoryRepository(db)

	authService := service.NewAuthService(cfg, userRepo, refreshTokenRepo, oneTimeTokenRepo, jwtManager)
	groupService := service.NewGroupService(groupRepo, groupMembershipRepo, groupInviteRepo, userRepo)
	phoneOTPService := service.NewPhoneOTPService(cfg, userRepo, phoneOTPRepo, authService)
	expenseService := service.NewExpenseService(groupRepo, groupMembershipRepo, expenseCategoryRepo, expenseRepo, expenseShareRepo)

	router := httptransport.NewRouter(httptransport.RouterDeps{
		Config:          cfg,
		Logger:          logger,
		JWTManager:      jwtManager,
		AuthHandler:     handlers.NewAuthHandler(authService),
		MeHandler:       handlers.NewMeHandler(userRepo),
		HealthHandler:   handlers.NewHealthHandler(db, nil),
		GroupHandler:    handlers.NewGroupHandler(groupService, cfg.FrontendURL),
		PhoneOTPHandler: handlers.NewPhoneOTPHandler(phoneOTPService),
		ExpenseHandler:  handlers.NewExpenseHandler(expenseService),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv.URL
}

type apiClient struct {
	baseURL string
	token   string
}

// do sends a JSON request and decodes the JSON response body (if any) into a
// generic map so tests can dig out whatever fields they need without a wall
// of one-off response structs.
func (c *apiClient) do(t *testing.T, method, path string, body any) (int, map[string]any) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var decoded map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &decoded)
	}
	return resp.StatusCode, decoded
}

// dig walks nested map[string]any values, failing the test loudly instead of
// panicking on a bad type assertion when a response shape changes.
func dig(t *testing.T, m map[string]any, path ...string) any {
	t.Helper()
	var cur any = m
	for i, key := range path {
		asMap, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("dig %v: %q is not an object (stopped at %q), got %#v", path, path[:i], key, cur)
		}
		cur, ok = asMap[key]
		if !ok {
			t.Fatalf("dig %v: missing key %q in %#v", path, key, asMap)
		}
	}
	return cur
}

func digStr(t *testing.T, m map[string]any, path ...string) string {
	t.Helper()
	v := dig(t, m, path...)
	s, ok := v.(string)
	if !ok {
		t.Fatalf("dig %v: expected string, got %#v", path, v)
	}
	return s
}

var userCounter int

func uniqueEmail(t *testing.T) string {
	userCounter++
	return fmt.Sprintf("test.%s.%d@example.com", t.Name(), userCounter)
}

// signupUser creates a fresh account and returns an authenticated client plus
// the new user's ID. Signup issues real tokens even though the frontend
// deliberately discards them and sends the user to /login instead.
func signupUser(t *testing.T, base, fullName string) (*apiClient, string) {
	t.Helper()
	c := &apiClient{baseURL: base}

	status, resp := c.do(t, http.MethodPost, "/api/v1/auth/signup", map[string]any{
		"full_name": fullName,
		"email":     uniqueEmail(t),
		"password":  "Sup3rSecret!Pass123",
	})
	if status != http.StatusCreated {
		t.Fatalf("signup %s: expected 201, got %d body=%v", fullName, status, resp)
	}

	data := resp["data"].(map[string]any)
	userID := digStr(t, data, "user", "id")
	c.token = digStr(t, data, "tokens", "access_token")
	return c, userID
}

func createGroup(t *testing.T, c *apiClient, name string) string {
	t.Helper()
	status, resp := c.do(t, http.MethodPost, "/api/v1/groups", map[string]any{"name": name})
	if status != http.StatusCreated {
		t.Fatalf("create group: expected 201, got %d body=%v", status, resp)
	}
	return digStr(t, resp["data"].(map[string]any), "group", "id")
}

func inviteAndJoin(t *testing.T, owner, joiner *apiClient, groupID string) {
	t.Helper()
	status, resp := owner.do(t, http.MethodPost, "/api/v1/groups/"+groupID+"/invites", nil)
	if status != http.StatusCreated {
		t.Fatalf("create invite: expected 201, got %d body=%v", status, resp)
	}
	code := digStr(t, resp["data"].(map[string]any), "invite", "invite_code")

	status, resp = joiner.do(t, http.MethodPost, "/api/v1/groups/join", map[string]any{"invite_code": code})
	if status != http.StatusOK {
		t.Fatalf("join group: expected 200, got %d body=%v", status, resp)
	}
}

// --- Auth ---------------------------------------------------------------

func TestAuthFlow(t *testing.T) {
	base := newTestServer(t)

	t.Run("signup then get me", func(t *testing.T) {
		c, userID := signupUser(t, base, "Ada Lovelace")
		status, resp := c.do(t, http.MethodGet, "/api/v1/users/me", nil)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", status, resp)
		}
		if got := digStr(t, resp["data"].(map[string]any), "user", "id"); got != userID {
			t.Fatalf("expected user id %s, got %s", userID, got)
		}
	})

	t.Run("get me without token is rejected", func(t *testing.T) {
		c := &apiClient{baseURL: base}
		status, _ := c.do(t, http.MethodGet, "/api/v1/users/me", nil)
		if status != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", status)
		}
	})

	t.Run("duplicate signup email is rejected", func(t *testing.T) {
		c := &apiClient{baseURL: base}
		email := uniqueEmail(t)
		status, _ := c.do(t, http.MethodPost, "/api/v1/auth/signup", map[string]any{
			"full_name": "First User",
			"email":     email,
			"password":  "Sup3rSecret!Pass123",
		})
		if status != http.StatusCreated {
			t.Fatalf("first signup: expected 201, got %d", status)
		}
		status, resp := c.do(t, http.MethodPost, "/api/v1/auth/signup", map[string]any{
			"full_name": "Second User",
			"email":     email,
			"password":  "Sup3rSecret!Pass123",
		})
		if status != http.StatusConflict {
			t.Fatalf("duplicate signup: expected 409, got %d body=%v", status, resp)
		}
	})

	t.Run("login with wrong password is rejected", func(t *testing.T) {
		c := &apiClient{baseURL: base}
		email := uniqueEmail(t)
		status, _ := c.do(t, http.MethodPost, "/api/v1/auth/signup", map[string]any{
			"full_name": "Wrong Password User",
			"email":     email,
			"password":  "Sup3rSecret!Pass123",
		})
		if status != http.StatusCreated {
			t.Fatalf("signup: expected 201, got %d", status)
		}
		status, resp := c.do(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
			"email":    email,
			"password": "TotallyWrongPassword1",
		})
		if status != http.StatusUnauthorized {
			t.Fatalf("login wrong password: expected 401, got %d body=%v", status, resp)
		}
	})
}

// --- Groups ---------------------------------------------------------------

func TestGroupsFlow(t *testing.T) {
	base := newTestServer(t)

	owner, ownerID := signupUser(t, base, "Group Owner")
	member, memberID := signupUser(t, base, "Group Member")
	outsider, _ := signupUser(t, base, "Outsider")

	groupID := createGroup(t, owner, "Goa Trip")
	inviteAndJoin(t, owner, member, groupID)

	t.Run("members list has both owner and member with correct roles", func(t *testing.T) {
		status, resp := owner.do(t, http.MethodGet, "/api/v1/groups/"+groupID+"/members", nil)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", status, resp)
		}
		members := resp["data"].(map[string]any)["members"].([]any)
		if len(members) != 2 {
			t.Fatalf("expected 2 members, got %d: %v", len(members), members)
		}
		roles := map[string]string{}
		for _, raw := range members {
			m := raw.(map[string]any)
			roles[m["user_id"].(string)] = m["role"].(string)
		}
		if roles[ownerID] != "owner" {
			t.Fatalf("expected owner role for %s, got %q", ownerID, roles[ownerID])
		}
		if roles[memberID] != "member" {
			t.Fatalf("expected member role for %s, got %q", memberID, roles[memberID])
		}
	})

	t.Run("outsider cannot view the group", func(t *testing.T) {
		status, _ := outsider.do(t, http.MethodGet, "/api/v1/groups/"+groupID, nil)
		if status != http.StatusNotFound {
			t.Fatalf("expected 404 for non-member access, got %d", status)
		}
	})

	t.Run("joining with a garbage invite code is rejected", func(t *testing.T) {
		status, resp := outsider.do(t, http.MethodPost, "/api/v1/groups/join", map[string]any{"invite_code": "NOTAREALCODE"})
		if status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%v", status, resp)
		}
	})
}

// --- Expenses ---------------------------------------------------------------

// expenseTestGroup sets up a group with three active members (owner + two
// invited members) plus a fourth user who is deliberately left out, for
// testing permission and membership edge cases.
type expenseTestGroup struct {
	groupID          string
	owner, memberB, memberC, outsider *apiClient
	ownerID, memberBID, memberCID string
}

func setupExpenseTestGroup(t *testing.T, base string) expenseTestGroup {
	t.Helper()
	owner, ownerID := signupUser(t, base, "Payer Owner")
	memberB, memberBID := signupUser(t, base, "Member B")
	memberC, memberCID := signupUser(t, base, "Member C")
	outsider, _ := signupUser(t, base, "Outsider")

	groupID := createGroup(t, owner, "Trip Expenses")
	inviteAndJoin(t, owner, memberB, groupID)
	inviteAndJoin(t, owner, memberC, groupID)

	return expenseTestGroup{
		groupID: groupID, owner: owner, memberB: memberB, memberC: memberC, outsider: outsider,
		ownerID: ownerID, memberBID: memberBID, memberCID: memberCID,
	}
}

func sumNetBalances(t *testing.T, resp map[string]any) float64 {
	t.Helper()
	balances := resp["data"].(map[string]any)["balances"].([]any)
	var sum float64
	for _, raw := range balances {
		sum += raw.(map[string]any)["net"].(float64)
	}
	return sum
}

func netByUser(t *testing.T, resp map[string]any) map[string]float64 {
	t.Helper()
	balances := resp["data"].(map[string]any)["balances"].([]any)
	out := map[string]float64{}
	for _, raw := range balances {
		m := raw.(map[string]any)
		out[m["user_id"].(string)] = m["net"].(float64)
	}
	return out
}

func TestExpensesFlow(t *testing.T) {
	base := newTestServer(t)

	t.Run("equal split among three members balances to zero", func(t *testing.T) {
		g := setupExpenseTestGroup(t, base)

		status, resp := g.owner.do(t, http.MethodPost, "/api/v1/groups/"+g.groupID+"/expenses", map[string]any{
			"title":          "Dinner",
			"amount":         100.00,
			"category_slug":  "food",
			"paid_by_user_id": g.ownerID,
			"split_method":   "equal",
			"expense_date":   "2026-08-18",
			"participants": []map[string]any{
				{"user_id": g.ownerID}, {"user_id": g.memberBID}, {"user_id": g.memberCID},
			},
		})
		if status != http.StatusCreated {
			t.Fatalf("create expense: expected 201, got %d body=%v", status, resp)
		}
		if slug := digStr(t, resp["data"].(map[string]any), "expense", "category_slug"); slug != "food" {
			t.Fatalf("expected category_slug 'food' to round-trip, got %q", slug)
		}

		status, balResp := g.owner.do(t, http.MethodGet, "/api/v1/groups/"+g.groupID+"/balances", nil)
		if status != http.StatusOK {
			t.Fatalf("get balances: expected 200, got %d body=%v", status, balResp)
		}
		if sum := sumNetBalances(t, balResp); sum > 0.01 || sum < -0.01 {
			t.Fatalf("balances must net to zero (money conservation), got sum=%v", sum)
		}
		net := netByUser(t, balResp)
		if net[g.ownerID] <= 0 {
			t.Fatalf("owner paid the whole bill, expected a positive net, got %v", net[g.ownerID])
		}
		if net[g.memberBID] >= 0 || net[g.memberCID] >= 0 {
			t.Fatalf("members B and C should owe money, got B=%v C=%v", net[g.memberBID], net[g.memberCID])
		}
	})

	t.Run("unequal split must sum to the total", func(t *testing.T) {
		g := setupExpenseTestGroup(t, base)

		status, resp := g.owner.do(t, http.MethodPost, "/api/v1/groups/"+g.groupID+"/expenses", map[string]any{
			"title": "Taxi", "amount": 90.00, "paid_by_user_id": g.ownerID, "split_method": "unequal",
			"expense_date": "2026-08-18",
			"participants": []map[string]any{
				{"user_id": g.ownerID, "amount": 30.00},
				{"user_id": g.memberBID, "amount": 30.00},
				{"user_id": g.memberCID, "amount": 29.00}, // 89 total, should fail
			},
		})
		if status != http.StatusBadRequest {
			t.Fatalf("mismatched unequal split: expected 400, got %d body=%v", status, resp)
		}

		status, resp = g.owner.do(t, http.MethodPost, "/api/v1/groups/"+g.groupID+"/expenses", map[string]any{
			"title": "Taxi", "amount": 90.00, "paid_by_user_id": g.ownerID, "split_method": "unequal",
			"expense_date": "2026-08-18",
			"participants": []map[string]any{
				{"user_id": g.ownerID, "amount": 30.00},
				{"user_id": g.memberBID, "amount": 30.00},
				{"user_id": g.memberCID, "amount": 30.00},
			},
		})
		if status != http.StatusCreated {
			t.Fatalf("correct unequal split: expected 201, got %d body=%v", status, resp)
		}
	})

	t.Run("percentage split must sum to exactly 100", func(t *testing.T) {
		g := setupExpenseTestGroup(t, base)

		// This is the case a naive frontend tolerance check would wave through
		// (99.99 is within 0.01 of 100) but the backend correctly rejects,
		// since 33.33+33.33+33.33 is 99.99, not 100.
		status, resp := g.owner.do(t, http.MethodPost, "/api/v1/groups/"+g.groupID+"/expenses", map[string]any{
			"title": "Hotel", "amount": 300.00, "paid_by_user_id": g.ownerID, "split_method": "percentage",
			"expense_date": "2026-08-18",
			"participants": []map[string]any{
				{"user_id": g.ownerID, "percentage": 33.33},
				{"user_id": g.memberBID, "percentage": 33.33},
				{"user_id": g.memberCID, "percentage": 33.33},
			},
		})
		if status != http.StatusBadRequest {
			t.Fatalf("percentages summing to 99.99%%: expected 400, got %d body=%v", status, resp)
		}

		status, resp = g.owner.do(t, http.MethodPost, "/api/v1/groups/"+g.groupID+"/expenses", map[string]any{
			"title": "Hotel", "amount": 300.00, "paid_by_user_id": g.ownerID, "split_method": "percentage",
			"expense_date": "2026-08-18",
			"participants": []map[string]any{
				{"user_id": g.ownerID, "percentage": 33.34},
				{"user_id": g.memberBID, "percentage": 33.33},
				{"user_id": g.memberCID, "percentage": 33.33},
			},
		})
		if status != http.StatusCreated {
			t.Fatalf("percentages summing to exactly 100%%: expected 201, got %d body=%v", status, resp)
		}
	})

	t.Run("shares split allocates proportionally with correct rounding", func(t *testing.T) {
		g := setupExpenseTestGroup(t, base)

		status, resp := g.owner.do(t, http.MethodPost, "/api/v1/groups/"+g.groupID+"/expenses", map[string]any{
			"title": "Groceries", "amount": 100.00, "paid_by_user_id": g.ownerID, "split_method": "shares",
			"expense_date": "2026-08-18",
			"participants": []map[string]any{
				{"user_id": g.ownerID, "shares": 1},
				{"user_id": g.memberBID, "shares": 1},
				{"user_id": g.memberCID, "shares": 1},
			},
		})
		if status != http.StatusCreated {
			t.Fatalf("create shares expense: expected 201, got %d body=%v", status, resp)
		}

		_, balResp := g.owner.do(t, http.MethodGet, "/api/v1/groups/"+g.groupID+"/balances", nil)
		if sum := sumNetBalances(t, balResp); sum > 0.01 || sum < -0.01 {
			t.Fatalf("1:1:1 shares on 100.00 must still net to zero, got sum=%v", sum)
		}
	})

	t.Run("participant who is not a group member is rejected", func(t *testing.T) {
		g := setupExpenseTestGroup(t, base)
		_, outsiderID := signupUser(t, base, "Random Non-Member")

		status, resp := g.owner.do(t, http.MethodPost, "/api/v1/groups/"+g.groupID+"/expenses", map[string]any{
			"title": "Snacks", "amount": 20.00, "paid_by_user_id": g.ownerID, "split_method": "equal",
			"expense_date": "2026-08-18",
			"participants": []map[string]any{
				{"user_id": g.ownerID}, {"user_id": outsiderID},
			},
		})
		if status != http.StatusBadRequest {
			t.Fatalf("non-member participant: expected 400, got %d body=%v", status, resp)
		}
	})

	t.Run("non-member cannot create an expense in the group", func(t *testing.T) {
		g := setupExpenseTestGroup(t, base)
		status, resp := g.outsider.do(t, http.MethodPost, "/api/v1/groups/"+g.groupID+"/expenses", map[string]any{
			"title": "Snacks", "amount": 20.00, "paid_by_user_id": g.ownerID, "split_method": "equal",
			"expense_date": "2026-08-18",
			"participants": []map[string]any{{"user_id": g.ownerID}},
		})
		if status != http.StatusNotFound {
			t.Fatalf("outsider creating expense: expected 404, got %d body=%v", status, resp)
		}
	})

	t.Run("delete permissions and soft-delete correctness", func(t *testing.T) {
		g := setupExpenseTestGroup(t, base)

		status, resp := g.memberB.do(t, http.MethodPost, "/api/v1/groups/"+g.groupID+"/expenses", map[string]any{
			"title": "Museum tickets", "amount": 60.00, "paid_by_user_id": g.memberBID, "split_method": "equal",
			"expense_date": "2026-08-18",
			"participants": []map[string]any{{"user_id": g.memberBID}, {"user_id": g.memberCID}},
		})
		if status != http.StatusCreated {
			t.Fatalf("create expense: expected 201, got %d body=%v", status, resp)
		}
		expenseID := digStr(t, resp["data"].(map[string]any), "expense", "id")

		// memberC is a participant but neither the creator, the payer, nor an
		// owner/admin, so deletion must be forbidden.
		status, resp = g.memberC.do(t, http.MethodDelete, "/api/v1/expenses/"+expenseID, nil)
		if status != http.StatusForbidden {
			t.Fatalf("uninvolved member deleting expense: expected 403, got %d body=%v", status, resp)
		}

		// The group owner may still delete any expense in their group.
		status, _ = g.owner.do(t, http.MethodDelete, "/api/v1/expenses/"+expenseID, nil)
		if status != http.StatusNoContent {
			t.Fatalf("owner deleting expense: expected 204, got %d", status)
		}

		status, listResp := g.owner.do(t, http.MethodGet, "/api/v1/groups/"+g.groupID+"/expenses", nil)
		if status != http.StatusOK {
			t.Fatalf("list expenses: expected 200, got %d", status)
		}
		for _, raw := range listResp["data"].(map[string]any)["expenses"].([]any) {
			if raw.(map[string]any)["id"].(string) == expenseID {
				t.Fatalf("deleted expense %s still appears in the list", expenseID)
			}
		}

		status, balResp := g.owner.do(t, http.MethodGet, "/api/v1/groups/"+g.groupID+"/balances", nil)
		if status != http.StatusOK {
			t.Fatalf("get balances: expected 200, got %d", status)
		}
		if sum := sumNetBalances(t, balResp); sum > 0.01 || sum < -0.01 {
			t.Fatalf("balances after delete must still net to zero, got sum=%v", sum)
		}
		net := netByUser(t, balResp)
		if net[g.memberBID] != 0 || net[g.memberCID] != 0 {
			t.Fatalf("deleted expense must not affect balances, got B=%v C=%v", net[g.memberBID], net[g.memberCID])
		}
	})

	t.Run("expense categories are seeded and listable", func(t *testing.T) {
		g := setupExpenseTestGroup(t, base)
		status, resp := g.owner.do(t, http.MethodGet, "/api/v1/expense-categories", nil)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", status, resp)
		}
		categories := resp["data"].(map[string]any)["categories"].([]any)
		if len(categories) == 0 {
			t.Fatalf("expected seeded categories, got none")
		}
		found := false
		for _, raw := range categories {
			if raw.(map[string]any)["slug"] == "food" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected 'food' category to be seeded, categories=%v", categories)
		}
	})
}
