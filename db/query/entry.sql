-- name: CreateEntry :one
INSERT INTO entries (
  account_id,
  amount
) VALUES (
  $1, $2
)
RETURNING *; 


-- name: ListEntries :many
SELECT * FROM entries
WHERE account_id = $1
LIMIT $2
OFFSET $3;