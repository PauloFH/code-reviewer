package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func (r *SQLiteRepository) ClaimReview(clientName, projectRef string, mrIID, mrID int64, username string) (bool, error) {
	result, err := r.db.Exec(`
		INSERT OR IGNORE INTO reviewed_merge_requests
			(client_name, project_ref, mr_iid, mr_id, reviewed_user, reviewed_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		clientName, projectRef, mrIID, mrID, username, now(),
	)
	if err != nil {
		return false, fmt.Errorf("claiming review: %w", err)
	}
	claimed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("checking review claim: %w", err)
	}
	return claimed == 1, nil
}

func (r *SQLiteRepository) GetReviewHistory(clientName, projectRef string, mrIID int64, username string) (*domain.ReviewHistory, error) {
	var history domain.ReviewHistory
	err := r.db.QueryRow(`SELECT review_head_sha, review_body FROM reviewed_merge_requests WHERE client_name=? AND project_ref=? AND mr_iid=? AND reviewed_user=?`,
		clientName, projectRef, mrIID, username).Scan(&history.HeadSHA, &history.Body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting review history: %w", err)
	}
	if history.Body == "" {
		return nil, nil
	}
	return &history, nil
}

func (r *SQLiteRepository) CompleteReview(clientName, projectRef string, mrIID int64, username, headSHA, body string) error {
	_, err := r.db.Exec(`UPDATE reviewed_merge_requests SET review_head_sha=?, review_body=? WHERE client_name=? AND project_ref=? AND mr_iid=? AND reviewed_user=?`,
		headSHA, body, clientName, projectRef, mrIID, username)
	if err != nil {
		return fmt.Errorf("saving review history: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ClearReview(clientName, projectRef string, mrIID int64) error {
	_, err := r.db.Exec(`DELETE FROM reviewed_merge_requests WHERE client_name=? AND project_ref=? AND mr_iid=?`,
		clientName, projectRef, mrIID)
	if err != nil {
		return fmt.Errorf("clearing review: %w", err)
	}
	return nil
}
