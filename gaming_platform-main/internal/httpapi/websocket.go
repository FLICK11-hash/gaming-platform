package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gaming_platform/internal/auth"
	"gaming_platform/internal/domain/match"
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
	defer s.matchSvc.Leave(ctx, matchID, userID, conn)

	for {
		var msg inboundWSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		switch msg.Type {
		case "move":
			input := match.MoveInput{From: msg.From, To: msg.To, Promotion: msg.Promotion, Column: msg.Column, IsPremove: msg.Premove}
			if err := s.matchSvc.HandleMove(ctx, matchID, userID, input); err != nil {
				s.matchSvc.SendError(matchID, userID, err.Error())
			}
		case "resign":
			if err := s.matchSvc.HandleResign(ctx, matchID, userID); err != nil {
				s.matchSvc.SendError(matchID, userID, err.Error())
			}
		default:
			s.matchSvc.SendError(matchID, userID, "unknown message type")
		}
	}
}
