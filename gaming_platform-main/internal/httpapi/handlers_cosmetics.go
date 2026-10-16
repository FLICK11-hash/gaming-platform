package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"gaming_platform/internal/auth"
	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
)

// starterAvatarImageKey is the one cosmetic every user starts with
// (users.selected_avatar defaults to it, see migration 0002_virtual_coins)
// without ever getting an explicit user_cosmetics row. It's the only
// coin_price = 0 item that should be treated as always-owned: the other
// coin_price = 0 items (mission/leaderboard/achievement rewards) must
// actually be earned before they're owned or equippable.
const starterAvatarImageKey = "rook"

// handleListCosmetics returns the active cosmetic catalog, flagging which
// items the caller already owns.
func (s *Server) handleListCosmetics(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	q := db.New(s.pool)

	items, err := q.ListActiveCosmetics(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list cosmetics")
		return
	}

	ownedIDs, err := q.ListUserCosmeticItemIDs(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list cosmetics")
		return
	}
	owned := make(map[int64]bool, len(ownedIDs))
	for _, id := range ownedIDs {
		owned[id] = true
	}

	resp := make([]cosmeticItemResponse, 0, len(items))
	for _, item := range items {
		isOwned := owned[item.CosmeticItemID] || (item.CoinPrice == 0 && item.ImageKey == starterAvatarImageKey)
		resp = append(resp, toCosmeticItemResponse(item, isOwned))
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleBuyCosmetic debits the item's coin price and grants ownership,
// atomically, so a failed grant can never take a player's coins.
func (s *Server) handleBuyCosmetic(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	cosmeticItemID, err := strconv.ParseInt(chi.URLParam(r, "cosmeticItemID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid cosmetic item id")
		return
	}

	q := db.New(s.pool)

	item, err := q.GetCosmeticItem(r.Context(), cosmeticItemID)
	if err != nil {
		writeError(w, http.StatusNotFound, "cosmetic item not found")
		return
	}
	if !item.IsActive {
		writeError(w, http.StatusBadRequest, "cosmetic item is not available")
		return
	}
	if item.CoinPrice == 0 {
		writeError(w, http.StatusBadRequest, "this item cannot be purchased; it must be unlocked")
		return
	}

	owned, err := q.UserOwnsCosmetic(r.Context(), db.UserOwnsCosmeticParams{
		UserID:         userID,
		CosmeticItemID: cosmeticItemID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check ownership")
		return
	}
	if owned {
		writeError(w, http.StatusConflict, "cosmetic item already owned")
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to purchase cosmetic item")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := db.New(s.pool).WithTx(tx)

	if item.CoinPrice > 0 {
		if _, err := qtx.AdjustUserCoinBalance(r.Context(), db.AdjustUserCoinBalanceParams{
			UserID: userID,
			Delta:  -item.CoinPrice,
		}); err != nil {
			if err == pgx.ErrNoRows {
				writeError(w, http.StatusBadRequest, "insufficient coin balance")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to purchase cosmetic item")
			return
		}

		if _, err := qtx.CreateCoinTransaction(r.Context(), db.CreateCoinTransactionParams{
			UserID:          userID,
			Amount:          -item.CoinPrice,
			TransactionType: "cosmetic_purchase",
			Description:     dbutil.TextToPGText("Purchased " + item.Name),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to purchase cosmetic item")
			return
		}
	}

	if err := qtx.GrantCosmeticToUser(r.Context(), db.GrantCosmeticToUserParams{
		UserID:         userID,
		CosmeticItemID: cosmeticItemID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to purchase cosmetic item")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to purchase cosmetic item")
		return
	}

	writeJSON(w, http.StatusOK, toCosmeticItemResponse(item, true))
}

// ownsOrFreeCosmetic looks up a cosmetic item and confirms it is either free
// or already owned by userID before it can be equipped.
func (s *Server) ownsOrFreeCosmetic(r *http.Request, q *db.Queries, userID, cosmeticItemID int64, wantType string) (db.CosmeticItem, error) {
	item, err := q.GetCosmeticItem(r.Context(), cosmeticItemID)
	if err != nil {
		return db.CosmeticItem{}, err
	}
	if item.ItemType != wantType {
		return db.CosmeticItem{}, pgx.ErrNoRows
	}
	if item.CoinPrice == 0 && item.ImageKey == starterAvatarImageKey {
		return item, nil
	}
	owned, err := q.UserOwnsCosmetic(r.Context(), db.UserOwnsCosmeticParams{
		UserID:         userID,
		CosmeticItemID: cosmeticItemID,
	})
	if err != nil {
		return db.CosmeticItem{}, err
	}
	if !owned {
		return db.CosmeticItem{}, pgx.ErrNoRows
	}
	return item, nil
}

func (s *Server) handleEquipAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req equipCosmeticRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	q := db.New(s.pool)

	item, err := s.ownsOrFreeCosmetic(r, q, userID, req.CosmeticItemID, "avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "you do not own that avatar")
		return
	}

	user, err := q.SetSelectedAvatar(r.Context(), db.SetSelectedAvatarParams{
		UserID:         userID,
		SelectedAvatar: item.ImageKey,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to equip avatar")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func (s *Server) handleEquipBadge(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req equipCosmeticRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	q := db.New(s.pool)

	// A zero id clears the equipped badge.
	var badgeKey pgtype.Text
	if req.CosmeticItemID != 0 {
		item, err := s.ownsOrFreeCosmetic(r, q, userID, req.CosmeticItemID, "badge")
		if err != nil {
			writeError(w, http.StatusBadRequest, "you do not own that badge")
			return
		}
		badgeKey = dbutil.TextToPGText(item.ImageKey)
	}

	user, err := q.SetSelectedBadge(r.Context(), db.SetSelectedBadgeParams{
		UserID:        userID,
		SelectedBadge: badgeKey,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to equip badge")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// handleToggleProfileFrame equips a comedic profile frame the player owns if
// it isn't already equipped, or unequips it if it is. Up to
// MaxSelectedProfileFrames may be equipped at once, enforced atomically by
// the users_max_five_profile_frames CHECK constraint.
func (s *Server) handleToggleProfileFrame(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req equipCosmeticRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	q := db.New(s.pool)

	item, err := s.ownsOrFreeCosmetic(r, q, userID, req.CosmeticItemID, "profile_frame")
	if err != nil {
		writeError(w, http.StatusBadRequest, "you do not own that profile frame")
		return
	}

	user, err := q.ToggleSelectedProfileFrame(r.Context(), db.ToggleSelectedProfileFrameParams{
		UserID:   userID,
		ImageKey: item.ImageKey,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.CheckViolation {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("you can only equip up to %d profile frames at once", MaxSelectedProfileFrames))
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update profile frame")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}
