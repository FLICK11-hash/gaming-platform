-- name: AdjustUserCoinBalance :one
-- Applies a signed delta to a user's coin balance. The WHERE clause blocks
-- overdraft atomically (no separate check-then-debit race): a debit whose
-- magnitude exceeds the current balance matches zero rows and comes back as
-- pgx.ErrNoRows to the caller.
UPDATE users
SET coin_balance = coin_balance + sqlc.arg('delta')::bigint
WHERE user_id = sqlc.arg('user_id')
  AND coin_balance + sqlc.arg('delta')::bigint >= 0
RETURNING *;

-- name: CreateCoinTransaction :one
INSERT INTO coin_transactions (
    user_id,
    amount,
    transaction_type,
    description,
    related_match_id,
    related_tournament_id
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListCoinTransactionsForUser :many
SELECT * FROM coin_transactions
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;
