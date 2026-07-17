package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gaming_platform/internal/auth"
)

// handleMatchWS upgrades to a websocket and relays move/resign messages into
// the match service for the lifetime of the connection.
func (s *Server) handleMatchWS(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	matchID, err := strconv.ParseInt(chi.URLParam(r, "matchID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid match id")
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx := r.Context()

	if err := s.matchSvc.Join(ctx, matchID, userID, conn); err != nil {
		_ = conn.WriteJSON(map[string]string{"type": "error", "error": err.Error()})
		return
	}
	defer s.matchSvc.Leave(matchID, userID)

	for {
		var msg inboundWSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		switch msg.Type {
		case "move":
			if err := s.matchSvc.HandleMove(ctx, matchID, userID, msg.From, msg.To, msg.Promotion); err != nil {
				_ = conn.WriteJSON(map[string]string{"type": "error", "error": err.Error()})
			}
		case "resign":
			if err := s.matchSvc.HandleResign(ctx, matchID, userID); err != nil {
				_ = conn.WriteJSON(map[string]string{"type": "error", "error": err.Error()})
			}
		default:
			_ = conn.WriteJSON(map[string]string{"type": "error", "error": "unknown message type"})
		}
	}
}
