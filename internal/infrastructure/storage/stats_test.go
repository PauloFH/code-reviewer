package storage

import (
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func saveEvents(t *testing.T, repo *SQLiteRepository, events ...domain.ReviewEvent) {
	t.Helper()
	for _, e := range events {
		if e.Status == "" {
			e.Status = domain.ReviewStatusSuccess
		}
		e.ProjectRef = "p"
		if err := repo.SaveEvent(e); err != nil {
			t.Fatalf("SaveEvent: %v", err)
		}
	}
}

// O filtro de cliente do painel vale para os totais do topo, não só para a lista por cliente.
func TestGetStatsFiltersTotalsByClient(t *testing.T) {
	repo := newTestRepo(t)
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	saveEvents(t, repo,
		domain.ReviewEvent{ClientName: "acme", TotalTokens: 100, CostUSD: 1, ReviewedAt: yesterday},
		domain.ReviewEvent{ClientName: "acme", TotalTokens: 100, CostUSD: 1, ReviewedAt: yesterday},
		domain.ReviewEvent{ClientName: "globex", TotalTokens: 500, CostUSD: 5, ReviewedAt: yesterday},
		domain.ReviewEvent{ClientName: "acme", Status: domain.ReviewStatusError, ReviewedAt: yesterday},
	)
	stats, err := repo.GetStats(domain.ReviewEventFilter{ClientName: "acme"})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalReviews != 2 || stats.TotalTokens != 200 || stats.TotalCostUSD != 2 {
		t.Fatalf("totals not filtered by client: reviews=%d tokens=%d cost=%v", stats.TotalReviews, stats.TotalTokens, stats.TotalCostUSD)
	}
	if stats.ReviewsLast30d != 2 || stats.CostLast30d != 2 {
		t.Fatalf("last 30d not filtered by client: reviews=%d cost=%v", stats.ReviewsLast30d, stats.CostLast30d)
	}
	if len(stats.ByClient) != 1 || stats.ByClient[0].ClientName != "acme" {
		t.Fatalf("by client: %+v", stats.ByClient)
	}
	if all, _ := repo.GetStats(domain.ReviewEventFilter{}); all.TotalReviews != 3 || len(all.ByClient) != 2 {
		t.Fatalf("no client filter: reviews=%d byClient=%d", all.TotalReviews, len(all.ByClient))
	}
}

// The dashboard sends since as midnight of the chosen day: events later that
// same day must count.
func TestEventFiltersIncludeTheWholeSinceDay(t *testing.T) {
	repo := newTestRepo(t)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	saveEvents(t, repo,
		domain.ReviewEvent{ClientName: "c", ReviewedAt: day.Add(-time.Hour)},
		domain.ReviewEvent{ClientName: "c", ReviewedAt: day.Add(9 * time.Hour)},
		domain.ReviewEvent{ClientName: "c", Status: domain.ReviewStatusError, ReviewedAt: day.Add(10 * time.Hour)},
	)
	f := domain.ReviewEventFilter{Since: day, Until: day.Add(24 * time.Hour)}
	if stats, _ := repo.GetStats(f); stats.TotalReviews != 1 {
		t.Errorf("GetStats since same day = %d, want 1", stats.TotalReviews)
	}
	if n, _ := repo.CountEvents(f); n != 2 {
		t.Errorf("CountEvents since same day = %d, want 2", n)
	}
	f.Status = domain.ReviewStatusError
	events, err := repo.ListEvents(f)
	if err != nil || len(events) != 1 || !events[0].ReviewedAt.Equal(day.Add(10*time.Hour)) {
		t.Errorf("ListEvents status filter = %+v %v", events, err)
	}
}
