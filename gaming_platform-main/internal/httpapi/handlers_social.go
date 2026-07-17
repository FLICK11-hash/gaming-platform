package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gaming_platform/internal/auth"
	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
)

func (s *Server) handleFollow(w http.ResponseWriter, r *http.Request) {
	followerID, _ := auth.UserID(r.Context())

	followedID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if followedID == followerID {
		writeError(w, http.StatusBadRequest, "cannot follow yourself")
		return
	}

	q := db.New(s.pool)
	if err := q.CreateFollow(r.Context(), db.CreateFollowParams{
		FollowingUserID: followerID,
		FollowedUserID:  followedID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to follow user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUnfollow(w http.ResponseWriter, r *http.Request) {
	followerID, _ := auth.UserID(r.Context())

	followedID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	q := db.New(s.pool)
	if err := q.DeleteFollow(r.Context(), db.DeleteFollowParams{
		FollowingUserID: followerID,
		FollowedUserID:  followedID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to unfollow user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListFollowers(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	q := db.New(s.pool)
	follows, err := q.ListFollowers(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list followers")
		return
	}

	resp := make([]followResponse, 0, len(follows))
	for _, f := range follows {
		resp = append(resp, toFollowResponse(f))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleListFollowing(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	q := db.New(s.pool)
	follows, err := q.ListFollowing(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list following")
		return
	}

	resp := make([]followResponse, 0, len(follows))
	for _, f := range follows {
		resp = append(resp, toFollowResponse(f))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserID(r.Context())

	var req createPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	q := db.New(s.pool)
	post, err := q.CreatePost(r.Context(), db.CreatePostParams{
		UserID: userID,
		Title:  dbutil.TextToPGText(req.Title),
		Body:   dbutil.TextToPGText(req.Body),
		Status: dbutil.TextToPGText("published"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create post")
		return
	}

	writeJSON(w, http.StatusCreated, toPostResponse(post))
}

func (s *Server) handleGetPost(w http.ResponseWriter, r *http.Request) {
	postID, err := strconv.ParseInt(chi.URLParam(r, "postID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid post id")
		return
	}

	q := db.New(s.pool)
	post, err := q.GetPost(r.Context(), postID)
	if err != nil {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}

	writeJSON(w, http.StatusOK, toPostResponse(post))
}

func (s *Server) handleListPostsByUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	q := db.New(s.pool)
	posts, err := q.ListPostsByUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list posts")
		return
	}

	resp := make([]postResponse, 0, len(posts))
	for _, p := range posts {
		resp = append(resp, toPostResponse(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpdatePostStatus(w http.ResponseWriter, r *http.Request) {
	postID, err := strconv.ParseInt(chi.URLParam(r, "postID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid post id")
		return
	}

	var req updatePostStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}

	q := db.New(s.pool)
	post, err := q.UpdatePostStatus(r.Context(), db.UpdatePostStatusParams{
		PostID: postID,
		Status: dbutil.TextToPGText(req.Status),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update post")
		return
	}

	writeJSON(w, http.StatusOK, toPostResponse(post))
}
