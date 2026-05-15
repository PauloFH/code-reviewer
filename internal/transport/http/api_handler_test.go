package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type statsRepo struct{}

func (statsRepo) SaveEvent(domain.ReviewEvent) error { return nil }
func (statsRepo) ListEvents(domain.ReviewEventFilter) ([]domain.ReviewEvent, error) {
	return nil, nil
}
func (statsRepo) CountEvents(domain.ReviewEventFilter) (int64, error) { return 0, nil }
func (statsRepo) GetStats(string) (*domain.ReviewStats, error) {
	return &domain.ReviewStats{}, nil
}
func (statsRepo) GetDailyCosts(string, int) ([]domain.DailyCostPoint, error) {
	return nil, nil
}

func TestGetStatsReturnsEmptyByClientArray(t *testing.T) {
	handler := NewAPIHandler(statsRepo{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var stats domain.ReviewStats
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if stats.ByClient == nil {
		t.Fatal("ByClient is nil, want empty slice")
	}
}
