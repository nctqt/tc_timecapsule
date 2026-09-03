-- name: CreateCase :one
INSERT INTO cases (
    id, 
    title, 
    description, 
    created_at, 
    updated_at
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetCaseByID :one
SELECT * 
FROM cases
WHERE id = $1 
LIMIT 1;

-- name: ListCases :many
SELECT * 
FROM cases
ORDER BY created_at DESC;

-- name: DeleteCase :exec
DELETE FROM cases
WHERE id = $1;
