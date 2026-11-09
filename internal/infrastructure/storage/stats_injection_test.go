package storage

import (
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func seedSuccessEvents(t *testing.T, repo *SQLiteRepository, ats ...time.Time) {
	t.Helper()
	for _, at := range ats {
		if err := repo.SaveEvent(domain.ReviewEvent{ClientName: "c", ProjectRef: "p", Status: domain.ReviewStatusSuccess, ReviewedAt: at}); err != nil {
			t.Fatalf("SaveEvent: %v", err)
		}
	}
}

// since/until vêm da query string: precisam chegar ao SQL como dado, nunca como código.
func TestGetStatsTreatsSinceUntilAsData(t *testing.T) {
	repo := newTestRepo(t)
	now := time.Now().UTC()
	seedSuccessEvents(t, repo, now.AddDate(0, 0, -2), now.AddDate(0, 0, -1))

	// Concatenado, vira `reviewed_at <= '' OR '1'='1'` e conta tudo.
	stats, err := repo.GetStats("", "", "' OR '1'='1")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalReviews != 0 || len(stats.ByClient) != 0 {
		t.Fatalf("injected until changed the query: total=%d byClient=%d", stats.TotalReviews, len(stats.ByClient))
	}

	// Concatenado, o UNION injeta uma linha forjada.
	stats, err = repo.GetStats("", "x' UNION SELECT 999,999,999 --", "")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalReviews != 0 {
		t.Fatalf("injected since forged a row: total=%d", stats.TotalReviews)
	}
}

func TestGetStatsFiltersBySinceUntil(t *testing.T) {
	repo := newTestRepo(t)
	now := time.Now().UTC()
	seedSuccessEvents(t, repo, now.AddDate(0, 0, -10), now.AddDate(0, 0, -1))

	since := now.AddDate(0, 0, -5).Format(time.RFC3339)
	stats, err := repo.GetStats("", since, "")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalReviews != 1 {
		t.Fatalf("since filter: total=%d, want 1", stats.TotalReviews)
	}
	if all, _ := repo.GetStats("", "", ""); all.TotalReviews != 2 {
		t.Fatalf("no filter: total=%d, want 2", all.TotalReviews)
	}
}
