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
