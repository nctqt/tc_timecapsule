-- +goose Up
ALTER TABLE videos 
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'pending_review',
    ADD COLUMN ai_summary TEXT,
    ADD COLUMN raw_transcript TEXT,
    ADD COLUMN estimated_event_date TIMESTAMPTZ,
    ADD COLUMN transcript_processed_at TIMESTAMPTZ,
    ADD COLUMN summary_source VARCHAR(20) NOT NULL DEFAULT 'metadata';

CREATE INDEX idx_videos_status ON videos(status);

CREATE INDEX idx_videos_transcript_pending 
    ON videos(id) 
    WHERE transcript_processed_at IS NULL AND raw_transcript IS NOT NULL;

CREATE INDEX idx_videos_milestone_status_event_date 
ON videos (milestone_id, status, COALESCE(estimated_event_date, published_at) ASC);

-- +goose Down
DROP INDEX IF EXISTS idx_videos_milestone_status_event_date;
DROP INDEX IF EXISTS idx_videos_transcript_pending;
DROP INDEX IF EXISTS idx_videos_status;

ALTER TABLE videos 
    DROP COLUMN IF EXISTS summary_source,
    DROP COLUMN IF EXISTS transcript_processed_at,
    DROP COLUMN IF EXISTS estimated_event_date,
    DROP COLUMN IF EXISTS raw_transcript,
    DROP COLUMN IF EXISTS ai_summary,
    DROP COLUMN IF EXISTS status;