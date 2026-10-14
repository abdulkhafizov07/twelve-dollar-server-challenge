package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"
)

type Post struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	Author    string `json:"author"`
	LikeCount int64  `json:"like_count"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	DB      string `json:"db"`
	UptimeS int64  `json:"uptime_s"`
}

type HealthErrorResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
	Error  string `json:"error"`
}

type FeedResponse struct {
	Posts []Post `json:"posts"`
}

type PostResponse struct {
	Post Post `json:"post"`
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if err := db.PingContext(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, HealthErrorResponse{
			Status: "degraded",
			DB:     "unreachable",
			Error:  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		DB:      "ok",
		UptimeS: int64(time.Since(startedAt).Seconds()),
	})
}

func FeedHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(r.Context(), `
		SELECT p.id, p.body, p.created_at, u.username,
		       (SELECT COUNT(*) FROM likes l WHERE l.post_id = p.id)
		FROM posts p
		JOIN users u ON u.id = p.user_id
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT 20
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer rows.Close()

	posts := make([]Post, 0, 20)
	for rows.Next() {
		var post Post
		if err := rows.Scan(
			&post.ID, &post.Body, &post.CreatedAt, &post.Author, &post.LikeCount,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, FeedResponse{Posts: posts})
}

func GetPostHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePostID(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid post id")
		return
	}

	post, err := getPost(r, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, PostResponse{Post: post})
}

func getPost(r *http.Request, id int64) (Post, error) {
	var post Post
	err := db.QueryRowContext(r.Context(), `
		SELECT p.id, p.body, p.created_at, u.username,
		       (SELECT COUNT(*) FROM likes l WHERE l.post_id = p.id)
		FROM posts p
		JOIN users u ON u.id = p.user_id
		WHERE p.id = ?
	`, id).Scan(&post.ID, &post.Body, &post.CreatedAt, &post.Author, &post.LikeCount)
	return post, err
}

func parsePostID(value string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0
}

