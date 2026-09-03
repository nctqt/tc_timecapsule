-- name: CreateVideo :one
INSERT INTO videos (
    id,
    milestone_id,
    youtube_video_id,
    title,
    channel_name,
    published_at,
    created_at,
    updated_at,
    category
) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListVideosByMilestone :many
SELECT * 
FROM videos
WHERE milestone_id = $1
ORDER BY published_at ASC;

-- name: ListUnlinkedVideos :many
SELECT * 
FROM videos
WHERE milestone_id IS NULL
ORDER BY published_at DESC;

-- name: GetVideoByID :one
SELECT *
FROM videos
WHERE id = $1
LIMIT 1;

-- name: LinkVideoToMilestone :exec
UPDATE videos
SET milestone_id = $1,
    updated_at = $3
WHERE id = $2;

-- name: UnlinkVideoFromMilestone :exec
UPDATE videos
SET milestone_id = NULL,
    updated_at = $2
WHERE id = $1;