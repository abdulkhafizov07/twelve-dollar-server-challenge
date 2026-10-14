package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type CreatePostRequest struct {
	Body any `json:"body"`
}

type CreatePostResponsePost struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	Author    string `json:"author"`
	LikeCount int64  `json:"like_count"`
}

type CreatePostResponse struct {
	Post CreatePostResponsePost `json:"post"`
}

type LikeResponse struct {
	Liked        bool  `json:"liked"`
	AlreadyLiked bool  `json:"already_liked"`
	PostID       int64 `json:"post_id"`
}

func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}

	var request CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "malformed JSON body")
		return
	}

	body, ok := request.Body.(string)
	if !ok {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	body = strings.TrimSpace(body)
	if body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	if len([]rune(body)) > 500 {
		writeError(w, http.StatusBadRequest, "body must be at most 500 characters")
		return
	}

	var id int64
	var createdAt string
	err := db.QueryRowContext(r.Context(), `
		INSERT INTO posts (user_id, body)
		VALUES (?, ?)
		RETURNING id, created_at
	`, user.ID, body).Scan(&id, &createdAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, CreatePostResponse{
		Post: CreatePostResponsePost{
			ID:        id,
			Body:      body,
			CreatedAt: createdAt,
			Author:    user.Username,
			LikeCount: 0,
		},
	})
}

func LikePostHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}

	id, ok := parsePostID(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid post id")
		return
	}

	var exists int
	err := db.QueryRowContext(r.Context(),
		`SELECT 1 FROM posts WHERE id = ?`, id,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	result, err := db.ExecContext(r.Context(), `
		INSERT INTO likes (user_id, post_id)
		VALUES (?, ?)
		ON CONFLICT (user_id, post_id) DO NOTHING
	`, user.ID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	inserted, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	alreadyLiked := inserted == 0
	status := http.StatusCreated
	if alreadyLiked {
		status = http.StatusOK
	}

	writeJSON(w, status, LikeResponse{
		Liked:        true,
		AlreadyLiked: alreadyLiked,
		PostID:       id,
	})
}

