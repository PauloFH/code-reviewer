package storage

import (
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestPurgeEventsBeforeRemovesOnlyOldEvents(t *testing.T) {
	repo := newTestRepo(t)
	now := time.Now().UTC()
	for _, at := range []time.Time{now.AddDate(-2, 0, 0), now.AddDate(0, -13, 0), now.AddDate(0, -1, 0), now} {
		if err := repo.SaveEvent(domain.ReviewEvent{ClientName: "c", ProjectRef: "p", AuthorUsername: "ana", ReviewedAt: at}); err != nil {
			t.Fatalf("SaveEvent: %v", err)
		}
	}

	n, err := repo.PurgeEventsBefore(now.AddDate(0, -12, 0))
	if err != nil {
		t.Fatalf("PurgeEventsBefore: %v", err)
	}
	if n != 2 {
		t.Fatalf("purged %d events, want 2", n)
	}
	left, err := repo.CountEvents(domain.ReviewEventFilter{})
	if err != nil {
		t.Fatalf("CountEvents: %v", err)
	}
	if left != 2 {
		t.Fatalf("%d events left, want 2", left)
	}
}

func TestPurgeEventsBeforeClearsOldReviewBodyButKeepsClaim(t *testing.T) {
	repo := newTestRepo(t)
	if _, err := repo.ClaimReview("c", "p", 1, 1, "bot"); err != nil {
		t.Fatalf("ClaimReview: %v", err)
	}
	if err := repo.CompleteReview("c", "p", 1, "bot", "sha", "texto da revisão"); err != nil {
		t.Fatalf("CompleteReview: %v", err)
	}

	// Corte no passado: a claim recente mantém o texto.
	if _, err := repo.PurgeEventsBefore(time.Now().AddDate(-1, 0, 0)); err != nil {
		t.Fatalf("PurgeEventsBefore: %v", err)
	}
	if h, _ := repo.GetReviewHistory("c", "p", 1, "bot"); h == nil || h.Body != "texto da revisão" {
		t.Fatalf("recent review body was cleared: %+v", h)
	}

	// Corte no futuro: o texto some, mas a claim continua bloqueando duplicata.
	if _, err := repo.PurgeEventsBefore(time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("PurgeEventsBefore: %v", err)
	}
	// Sem corpo não há histórico: um commit novo nesse MR recebe revisão completa, não incremental.
	if h, _ := repo.GetReviewHistory("c", "p", 1, "bot"); h != nil {
		t.Fatalf("old review body kept: %+v", h)
	}
	if claimed, err := repo.ClaimReview("c", "p", 1, 1, "bot"); err != nil || claimed {
		t.Fatalf("claim was lost: claimed=%v err=%v", claimed, err)
	}
}
