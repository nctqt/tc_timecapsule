-- name: CreateVideo :one
INSERT INTO videos (
    id,
    milestone_id,
    youtube_video_id,
    title,
    channel_name,
    description,
    published_at,
    created_at,
    updated_at,
    category,
    status,
    raw_transcript
) 
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: UpdateVideoAnalysis :one
UPDATE videos
SET 
    category = $2,
    ai_summary = $3,
    estimated_event_date = $4,
    summary_source = $5,
    status = $6,
    updated_at = $7
WHERE id = $1
RETURNING *;

-- name: GetVideoByID :one
SELECT *
FROM videos
WHERE id = $1
LIMIT 1;

-- name: ListVideosByMilestoneIDs :many
SELECT * 
FROM videos
WHERE milestone_id = ANY(sqlc.slice('milestone_ids')::uuid[])
  AND status IN ('analyzed', 'approved')
ORDER BY COALESCE(estimated_event_date, published_at) ASC;

-- name: ListUnlinkedVideos :many
SELECT * 
FROM videos
WHERE milestone_id IS NULL
ORDER BY published_at DESC;

-- name: ListPendingVideos :many
SELECT *
FROM videos
WHERE status = 'pending_review'
ORDER BY created_at DESC;

-- name: LinkVideoToMilestone :exec
UPDATE videos
SET 
    milestone_id = $1,
    status = 'approved',
    updated_at = $2
WHERE id = $3;

-- name: UnlinkVideoFromMilestone :exec
UPDATE videos
SET milestone_id = NULL,
    updated_at = $2
WHERE id = $1;

-- name: UpdateVideoStatus :exec
UPDATE videos
SET status = $2,
    updated_at = $3
WHERE id = $1;

-- name: GetPendingTranscripts :many
SELECT * 
FROM videos
WHERE raw_transcript IS NOT NULL 
  AND transcript_processed_at IS NULL
ORDER BY created_at ASC
LIMIT $1;

-- name: UpdateVideoTranscript :exec
UPDATE videos
SET raw_transcript = $2,
    updated_at = $3
WHERE id = $1;