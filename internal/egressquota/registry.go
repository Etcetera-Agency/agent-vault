package egressquota

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Infisical/agent-vault/internal/broker"
	"github.com/Infisical/agent-vault/internal/store"
)

const implicitAccount = "__service__"

type requestLogStore interface {
	ListRequestLogs(ctx context.Context, opts store.ListRequestLogsOpts) ([]store.RequestLog, error)
}

type Registry struct {
	mu       sync.Mutex
	now      func() time.Time
	counters map[string]*counterState
	limits   map[string]*limitState
}

type counterState struct {
	dayKey     string
	dayCount   int
	monthKey   string
	monthCount int
}

type limitState struct {
	rpmHits  []time.Time
	inFlight int
}

type Decision struct {
	Service    string
	RetryAfter time.Duration
	Reason     string
}

type Reservation struct {
	reg      *Registry
	key      string
	service  string
	released bool
	recorded bool
}

type Snapshot struct {
	VaultID        string
	MatchedService string
	CredentialKeys []string
	CreatedAt      time.Time
	Status         int
}

func New() *Registry {
	return &Registry{
		now:      time.Now,
		counters: make(map[string]*counterState),
		limits:   make(map[string]*limitState),
	}
}

func (r *Registry) SeedFromRequestLogs(ctx context.Context, s requestLogStore, limit int) error {
	return r.loadFromRequestLogs(ctx, s, limit, false)
}

func (r *Registry) ReconcileFromRequestLogs(ctx context.Context, s requestLogStore, limit int) error {
	return r.loadFromRequestLogs(ctx, s, limit, true)
}

func RunReconcile(ctx context.Context, r *Registry, s requestLogStore, every time.Duration) {
	if r == nil || s == nil {
		return
	}
	if every <= 0 {
		every = 5 * time.Minute
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = r.ReconcileFromRequestLogs(ctx, s, 10000)
		}
	}
}

func (r *Registry) loadFromRequestLogs(ctx context.Context, s requestLogStore, limit int, replace bool) error {
	if limit <= 0 {
		limit = 10000
	}
	rows, err := s.ListRequestLogs(ctx, store.ListRequestLogsOpts{Limit: limit})
	if err != nil {
		return err
	}
	snapshots := make([]Snapshot, 0, len(rows))
	for _, row := range rows {
		snapshots = append(snapshots, Snapshot{
			VaultID:        row.VaultID,
			MatchedService: row.MatchedService,
			CredentialKeys: row.CredentialKeys,
			CreatedAt:      row.CreatedAt,
			Status:         row.Status,
		})
	}
	if replace {
		r.Reconcile(snapshots)
	} else {
		r.Seed(snapshots)
	}
	return nil
}

func (r *Registry) Seed(rows []Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	dayKey, monthKey := windowKeys(now)
	for _, row := range rows {
		if row.MatchedService == "" || row.Status <= 0 || row.CreatedAt.IsZero() {
			continue
		}
		key := quotaKey(row.VaultID, row.MatchedService, accountKey(row.CredentialKeys))
		state := r.counterForLocked(key, now)
		rowDay, rowMonth := windowKeys(row.CreatedAt)
		if rowDay == dayKey {
			state.dayCount++
		}
		if rowMonth == monthKey {
			state.monthCount++
		}
	}
}

func (r *Registry) Reconcile(rows []Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	next := make(map[string]*counterState)
	dayKey, monthKey := windowKeys(now)
	for _, row := range rows {
		if row.MatchedService == "" || row.Status <= 0 || row.CreatedAt.IsZero() {
			continue
		}
		key := quotaKey(row.VaultID, row.MatchedService, accountKey(row.CredentialKeys))
		state := next[key]
		if state == nil {
			state = &counterState{dayKey: dayKey, monthKey: monthKey}
			next[key] = state
		}
		rowDay, rowMonth := windowKeys(row.CreatedAt)
		if rowDay == dayKey {
			state.dayCount++
		}
		if rowMonth == monthKey {
			state.monthCount++
		}
	}
	r.counters = next
}

func (r *Registry) Reserve(ctx context.Context, vaultID string, svc broker.Service) (*Reservation, *Decision) {
	if svc.Quota == nil {
		return nil, nil
	}
	select {
	case <-ctx.Done():
		return nil, &Decision{Service: svc.Name, RetryAfter: time.Second, Reason: "context"}
	default:
	}

	keys := svc.Auth.CredentialKeys()
	key := quotaKey(vaultID, svc.Name, accountKey(keys))

	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	counter := r.counterForLocked(key, now)
	if svc.Quota.DailyCap != nil && counter.dayCount >= *svc.Quota.DailyCap {
		return nil, &Decision{Service: svc.Name, RetryAfter: untilNextDay(now), Reason: "daily_cap"}
	}
	if svc.Quota.MonthlyCap != nil && counter.monthCount >= *svc.Quota.MonthlyCap {
		return nil, &Decision{Service: svc.Name, RetryAfter: untilNextMonth(now), Reason: "monthly_cap"}
	}

	limit := r.limitForLocked(key)
	if svc.Quota.Concurrency != nil && limit.inFlight >= *svc.Quota.Concurrency {
		return nil, &Decision{Service: svc.Name, RetryAfter: time.Second, Reason: "concurrency"}
	}
	if svc.Quota.RPM != nil {
		retryAfter := reserveRPM(now, limit, *svc.Quota.RPM)
		if retryAfter > 0 {
			return nil, &Decision{Service: svc.Name, RetryAfter: retryAfter, Reason: "rpm"}
		}
	}
	if svc.Quota.Concurrency != nil {
		limit.inFlight++
	}
	return &Reservation{reg: r, key: key, service: svc.Name}, nil
}

func (r *Registry) Snapshot(vaultID, service string, credentialKeys []string) (dayCount, monthCount int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.counterForLocked(quotaKey(vaultID, service, accountKey(credentialKeys)), r.now())
	return state.dayCount, state.monthCount
}

func (r *Registry) counterForLocked(key string, now time.Time) *counterState {
	dayKey, monthKey := windowKeys(now)
	state := r.counters[key]
	if state == nil {
		state = &counterState{}
		r.counters[key] = state
	}
	if state.dayKey != dayKey {
		state.dayKey = dayKey
		state.dayCount = 0
	}
	if state.monthKey != monthKey {
		state.monthKey = monthKey
		state.monthCount = 0
	}
	return state
}

func (r *Registry) limitForLocked(key string) *limitState {
	state := r.limits[key]
	if state == nil {
		state = &limitState{}
		r.limits[key] = state
	}
	return state
}

func (res *Reservation) Commit(status int) {
	if res == nil || status <= 0 {
		return
	}
	res.reg.mu.Lock()
	defer res.reg.mu.Unlock()
	if res.recorded {
		return
	}
	state := res.reg.counterForLocked(res.key, res.reg.now())
	state.dayCount++
	state.monthCount++
	res.recorded = true
}

func (res *Reservation) Release() {
	if res == nil {
		return
	}
	res.reg.mu.Lock()
	defer res.reg.mu.Unlock()
	if res.released {
		return
	}
	if state := res.reg.limits[res.key]; state != nil && state.inFlight > 0 {
		state.inFlight--
	}
	res.released = true
}

func WriteDenial(w http.ResponseWriter, d *Decision) {
	retryAfter := int(d.RetryAfter.Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Vault-Quota-Exhausted", d.Service)
	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"error":"quota_exhausted","service":%q,"reason":%q}`+"\n", d.Service, d.Reason)))
}

func reserveRPM(now time.Time, state *limitState, rpm int) time.Duration {
	if rpm <= 0 {
		return 0
	}
	cutoff := now.Add(-time.Minute)
	kept := state.rpmHits[:0]
	for _, hit := range state.rpmHits {
		if hit.After(cutoff) {
			kept = append(kept, hit)
		}
	}
	state.rpmHits = kept
	if len(state.rpmHits) >= rpm {
		return state.rpmHits[0].Add(time.Minute).Sub(now)
	}
	state.rpmHits = append(state.rpmHits, now)
	return 0
}

func accountKey(keys []string) string {
	if len(keys) == 0 || keys[0] == "" {
		return implicitAccount
	}
	return keys[0]
}

func quotaKey(vaultID, service, account string) string {
	return vaultID + "\x00" + service + "\x00" + account
}

func windowKeys(t time.Time) (string, string) {
	utc := t.UTC()
	return utc.Format("2006-01-02"), utc.Format("2006-01")
}

func untilNextDay(t time.Time) time.Duration {
	utc := t.UTC()
	next := time.Date(utc.Year(), utc.Month(), utc.Day()+1, 0, 0, 0, 0, time.UTC)
	return next.Sub(utc)
}

func untilNextMonth(t time.Time) time.Duration {
	utc := t.UTC()
	next := time.Date(utc.Year(), utc.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return next.Sub(utc)
}
