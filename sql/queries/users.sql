-- name: CreateUser :one
INSERT INTO users (id, email, hashed_password, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * 
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT * 
FROM users
WHERE id = $1;

-- name: RecordVideoWatch :one
INSERT INTO watch_history (id, user_id, video_id, watched_at, completed)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, video_id) 
DO UPDATE SET 
    watched_at = EXCLUDED.watched_at,
    completed = EXCLUDED.completed
RETURNING *;

-- name: GetWatchHistoryByUserID :many
SELECT 
    wh.id AS watch_id,
    wh.watched_at,
    wh.completed,
    v.id AS video_id,
    v.youtube_video_id,
    v.title,
    v.channel_name
FROM watch_history wh
JOIN videos v ON wh.video_id = v.id
WHERE wh.user_id = $1
ORDER BY wh.watched_at DESC;