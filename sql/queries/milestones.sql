-- name: CreateMilestone :one
INSERT INTO milestones (
    id,
    case_id,
    title,
    description,
    event_date,
    date_precision,
    created_at,
    updated_at
) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListMilestonesByCase :many
SELECT * 
FROM milestones
WHERE case_id = $1
ORDER BY event_date ASC;