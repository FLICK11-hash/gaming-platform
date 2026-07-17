-- name: ListActiveCosmetics :many
SELECT * FROM cosmetic_items WHERE is_active = true ORDER BY item_type, coin_price;

-- name: GetCosmeticItem :one
SELECT * FROM cosmetic_items WHERE cosmetic_item_id = $1;

-- name: ListUserCosmeticItemIDs :many
SELECT cosmetic_item_id FROM user_cosmetics WHERE user_id = $1;

-- name: UserOwnsCosmetic :one
SELECT EXISTS(SELECT 1 FROM user_cosmetics WHERE user_id = $1 AND cosmetic_item_id = $2);

-- name: GrantCosmeticToUser :exec
INSERT INTO user_cosmetics (user_id, cosmetic_item_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
