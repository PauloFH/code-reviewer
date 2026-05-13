-- Migration: 001_create_reviewed_merge_requests
-- Tracks which MRs have already been reviewed per client and user,
-- guaranteeing that the automated review is posted at most once.

CREATE TABLE IF NOT EXISTS reviewed_merge_requests (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    client_name   TEXT     NOT NULL,
    project_ref   TEXT     NOT NULL,  -- numeric string (GitLab) or "owner/repo" (GitHub/Gitea)
    mr_iid        INTEGER  NOT NULL,  -- user-facing MR/PR number
    mr_id         INTEGER,            -- internal platform ID (best-effort)
    reviewed_user TEXT     NOT NULL,  -- username that triggered the review
    reviewed_at   DATETIME NOT NULL,

    UNIQUE(client_name, project_ref, mr_iid, reviewed_user)
);

CREATE INDEX IF NOT EXISTS idx_reviewed_lookup
    ON reviewed_merge_requests(client_name, project_ref, mr_iid, reviewed_user);
