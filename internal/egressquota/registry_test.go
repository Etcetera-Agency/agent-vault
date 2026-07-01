package egressquota

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Infisical/agent-vault/internal/broker"
	"github.com/Infisical/agent-vault/internal/store"
)

type fakeLogStore struct {
	rows []store.RequestLog
}

func (f fakeLogStore) ListRequestLogs(context.Context, store.ListRequestLogsOpts) ([]store.RequestLog, error) {
	return f.rows, nil
}

func intPtr(v int) *int { return &v }

func quotaService(q *broker.ServiceQuota) broker.Service {
	return broker.Service{
		Name:  "apify",
		Host:  "api.apify.com",
		Auth:  broker.Auth{Type: "bearer", Token: "APIFY_TOKEN"},
		Quota: q,
	}
}

func TestReserveDeniesDailyCap(t *testing.T) {
	reg := New()
	reg.Seed([]Snapshot{{
		VaultID:        "vault-1",
		MatchedService: "apify",
		CredentialKeys: []string{"APIFY_TOKEN"},
		CreatedAt:      time.Now(),
		Status:         http.StatusOK,
	}})

	_, denial := reg.Reserve(context.Background(), "vault-1", quotaService(&broker.ServiceQuota{DailyCap: intPtr(1)}))
	if denial == nil {
		t.Fatal("expected daily cap denial")
	}
	if denial.Service != "apify" || denial.Reason != "daily_cap" {
		t.Fatalf("denial = %+v, want apify daily_cap", denial)
	}
}

func TestReserveRecordsForwardedCall(t *testing.T) {
	reg := New()
	res, denial := reg.Reserve(context.Background(), "vault-1", quotaService(&broker.ServiceQuota{MonthlyCap: intPtr(2)}))
	if denial != nil {
		t.Fatalf("unexpected denial: %+v", denial)
	}
	res.Commit(http.StatusOK)
	res.Release()

	day, month := reg.Snapshot("vault-1", "apify", []string{"APIFY_TOKEN"})
	if day != 1 || month != 1 {
		t.Fatalf("counts = day %d month %d, want 1/1", day, month)
	}
}

func TestReserveDeniesRPM(t *testing.T) {
	reg := New()
	svc := quotaService(&broker.ServiceQuota{RPM: intPtr(1)})
	res, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected first denial: %+v", denial)
	}
	res.Release()

	_, denial = reg.Reserve(context.Background(), "vault-1", svc)
	if denial == nil || denial.Reason != "rpm" {
		t.Fatalf("denial = %+v, want rpm", denial)
	}
}

func TestReserveDeniesConcurrencyUntilRelease(t *testing.T) {
	reg := New()
	svc := quotaService(&broker.ServiceQuota{Concurrency: intPtr(1)})
	res, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected first denial: %+v", denial)
	}

	_, denial = reg.Reserve(context.Background(), "vault-1", svc)
	if denial == nil || denial.Reason != "concurrency" {
		t.Fatalf("denial = %+v, want concurrency", denial)
	}

	res.Release()
	res2, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("expected reserve after release, got %+v", denial)
	}
	res2.Release()
}

func TestReserveLeastUsedSelectsLowerUsageAccount(t *testing.T) {
	reg := New()
	reg.Seed([]Snapshot{
		{VaultID: "vault-1", MatchedService: "apify", AccountID: "acct1", CredentialKeys: []string{"APIFY_TOKEN_1"}, CreatedAt: time.Now(), Status: http.StatusOK},
		{VaultID: "vault-1", MatchedService: "apify", AccountID: "acct1", CredentialKeys: []string{"APIFY_TOKEN_1"}, CreatedAt: time.Now(), Status: http.StatusOK},
	})
	svc := quotaService(&broker.ServiceQuota{DailyCap: intPtr(10)})
	svc.Rotation = "least_used"
	svc.Accounts = []broker.ServiceAccount{
		{ID: "acct1", CredentialKey: "APIFY_TOKEN_1"},
		{ID: "acct2", CredentialKey: "APIFY_TOKEN_2"},
	}

	res, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected denial: %+v", denial)
	}
	if res.AccountID() != "acct2" || res.CredentialKey() != "APIFY_TOKEN_2" {
		t.Fatalf("reservation = account %q credential %q, want acct2/APIFY_TOKEN_2", res.AccountID(), res.CredentialKey())
	}
	res.Release()
}

func TestReserveRoundRobinAlternatesAccounts(t *testing.T) {
	reg := New()
	svc := quotaService(&broker.ServiceQuota{DailyCap: intPtr(10)})
	svc.Rotation = "round_robin"
	svc.Accounts = []broker.ServiceAccount{
		{ID: "acct1", CredentialKey: "APIFY_TOKEN_1"},
		{ID: "acct2", CredentialKey: "APIFY_TOKEN_2"},
	}

	first, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected first denial: %+v", denial)
	}
	first.Release()
	second, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected second denial: %+v", denial)
	}
	second.Release()
	if first.AccountID() != "acct1" || second.AccountID() != "acct2" {
		t.Fatalf("accounts = %q then %q, want acct1 then acct2", first.AccountID(), second.AccountID())
	}
}

func TestReserveRoundRobinAlternatesAccountOnlyService(t *testing.T) {
	reg := New()
	svc := quotaService(nil)
	svc.Rotation = "round_robin"
	svc.Accounts = []broker.ServiceAccount{
		{ID: "acct1", CredentialKey: "APIFY_TOKEN_1"},
		{ID: "acct2", CredentialKey: "APIFY_TOKEN_2"},
	}

	first, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected first denial: %+v", denial)
	}
	first.Commit(http.StatusOK)
	first.Release()
	second, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected second denial: %+v", denial)
	}
	second.Release()
	if first.AccountID() != "acct1" || second.AccountID() != "acct2" {
		t.Fatalf("accounts = %q then %q, want acct1 then acct2", first.AccountID(), second.AccountID())
	}
	day, month, _ := reg.Usage("vault-1", "apify", "acct1")
	if day != 1 || month != 1 {
		t.Fatalf("acct1 usage = day %d month %d, want 1/1", day, month)
	}
}

func TestReserveLeastUsedSelectsLowerUsageAccountOnlyService(t *testing.T) {
	reg := New()
	reg.Seed([]Snapshot{
		{VaultID: "vault-1", MatchedService: "apify", AccountID: "acct1", CredentialKeys: []string{"APIFY_TOKEN_1"}, CreatedAt: time.Now(), Status: http.StatusOK},
		{VaultID: "vault-1", MatchedService: "apify", AccountID: "acct1", CredentialKeys: []string{"APIFY_TOKEN_1"}, CreatedAt: time.Now(), Status: http.StatusOK},
	})
	svc := quotaService(nil)
	svc.Rotation = "least_used"
	svc.Accounts = []broker.ServiceAccount{
		{ID: "acct1", CredentialKey: "APIFY_TOKEN_1"},
		{ID: "acct2", CredentialKey: "APIFY_TOKEN_2"},
	}

	res, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected denial: %+v", denial)
	}
	if res.AccountID() != "acct2" {
		t.Fatalf("account = %q, want acct2", res.AccountID())
	}
	res.Release()
}

func TestReserveSkipsCooldownAccountOnlyService(t *testing.T) {
	reg := New()
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	reg.now = func() time.Time { return now }
	svc := quotaService(nil)
	svc.Rotation = "round_robin"
	svc.Accounts = []broker.ServiceAccount{
		{ID: "acct1", CredentialKey: "APIFY_TOKEN_1"},
		{ID: "acct2", CredentialKey: "APIFY_TOKEN_2"},
	}

	first, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected first denial: %+v", denial)
	}
	first.Cooldown(2 * time.Minute)
	first.Release()
	second, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected second denial: %+v", denial)
	}
	if second.AccountID() != "acct2" {
		t.Fatalf("account = %q, want acct2", second.AccountID())
	}
	second.Release()
}

func TestReserveDeniesWhenAllAccountOnlyAccountsCooling(t *testing.T) {
	reg := New()
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	reg.now = func() time.Time { return now }
	svc := quotaService(nil)
	svc.Rotation = "round_robin"
	svc.Accounts = []broker.ServiceAccount{
		{ID: "acct1", CredentialKey: "APIFY_TOKEN_1"},
		{ID: "acct2", CredentialKey: "APIFY_TOKEN_2"},
	}

	first, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected first denial: %+v", denial)
	}
	first.Cooldown(2 * time.Minute)
	first.Release()
	second, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected second denial: %+v", denial)
	}
	second.Cooldown(time.Minute)
	second.Release()

	_, denial = reg.Reserve(context.Background(), "vault-1", svc)
	if denial == nil || denial.Reason != "cooldown" {
		t.Fatalf("denial = %+v, want cooldown", denial)
	}
}

func TestReserveNoQuotaNoAccountsPassthrough(t *testing.T) {
	reg := New()
	res, denial := reg.Reserve(context.Background(), "vault-1", quotaService(nil))
	if res != nil || denial != nil {
		t.Fatalf("reservation/denial = %+v/%+v, want nil/nil", res, denial)
	}
}

func TestReserveSkipsCooldownAccount(t *testing.T) {
	reg := New()
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	reg.now = func() time.Time { return now }
	svc := quotaService(&broker.ServiceQuota{DailyCap: intPtr(10)})
	svc.Rotation = "round_robin"
	svc.Accounts = []broker.ServiceAccount{
		{ID: "acct1", CredentialKey: "APIFY_TOKEN_1"},
		{ID: "acct2", CredentialKey: "APIFY_TOKEN_2"},
	}

	first, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected first denial: %+v", denial)
	}
	first.Cooldown(2 * time.Minute)
	first.Release()

	second, denial := reg.Reserve(context.Background(), "vault-1", svc)
	if denial != nil {
		t.Fatalf("unexpected second denial: %+v", denial)
	}
	if second.AccountID() != "acct2" {
		t.Fatalf("account = %q, want acct2", second.AccountID())
	}
	second.Release()
}

func TestReserveEmitsSingleExhaustionAlertPerDay(t *testing.T) {
	reg := New()
	reg.now = func() time.Time { return time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC) }
	alerts := make(chan string, 2)
	reg.OnExhausted = func(_ context.Context, _, service string) {
		alerts <- service
	}
	svc := quotaService(&broker.ServiceQuota{DailyCap: intPtr(0)})

	for i := 0; i < 2; i++ {
		_, denial := reg.Reserve(context.Background(), "vault-1", svc)
		if denial == nil {
			t.Fatal("expected quota denial")
		}
	}

	select {
	case got := <-alerts:
		if got != "apify" {
			t.Fatalf("alert = %q, want apify", got)
		}
	case <-time.After(time.Second):
		t.Fatal("expected alert")
	}
	select {
	case got := <-alerts:
		t.Fatalf("unexpected duplicate alert %q", got)
	default:
	}
}

func TestSeedFromRequestLogsReconstructsWindows(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	reg := New()
	reg.now = func() time.Time { return now }
	err := reg.SeedFromRequestLogs(context.Background(), fakeLogStore{rows: []store.RequestLog{
		{VaultID: "vault-1", MatchedService: "apify", CredentialKeys: []string{"APIFY_TOKEN"}, CreatedAt: now, Status: http.StatusOK},
		{VaultID: "vault-1", MatchedService: "apify", CredentialKeys: []string{"APIFY_TOKEN"}, CreatedAt: now.AddDate(0, 0, -1), Status: http.StatusOK},
	}}, 100)
	if err != nil {
		t.Fatalf("SeedFromRequestLogs: %v", err)
	}

	day, month := reg.Snapshot("vault-1", "apify", []string{"APIFY_TOKEN"})
	if day != 1 || month != 2 {
		t.Fatalf("counts = day %d month %d, want 1/2", day, month)
	}
}

func TestSeedFromRequestLogsUsesAccountIDWhenCredentialKeyDiffers(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	reg := New()
	reg.now = func() time.Time { return now }
	err := reg.SeedFromRequestLogs(context.Background(), fakeLogStore{rows: []store.RequestLog{
		{VaultID: "vault-1", MatchedService: "apify", AccountID: "acct1", CredentialKeys: []string{"APIFY_TOKEN_PRIMARY"}, CreatedAt: now, Status: http.StatusOK},
	}}, 100)
	if err != nil {
		t.Fatalf("SeedFromRequestLogs: %v", err)
	}

	day, month, _ := reg.Usage("vault-1", "apify", "acct1")
	if day != 1 || month != 1 {
		t.Fatalf("account id counts = day %d month %d, want 1/1", day, month)
	}
	day, month, _ = reg.Usage("vault-1", "apify", "APIFY_TOKEN_PRIMARY")
	if day != 0 || month != 0 {
		t.Fatalf("credential key counts = day %d month %d, want 0/0", day, month)
	}
}
