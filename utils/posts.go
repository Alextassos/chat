package utils

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type Post struct {
	ID           int    `json:"id"`
	UserID       int    `json:"user_id"`
	Author       string `json:"author"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	Categories   string `json:"categories"`
	CreatedAt    string `json:"created_at"`
	Likes        int    `json:"likes"`
	Dislikes     int    `json:"dislikes"`
	UserReaction int    `json:"user_reaction"`
}

func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input Post
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	userID, _, err := GetUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	input.Title = trimAndNormalize(input.Title)
	input.Content = trimAndNormalize(input.Content)
	if input.Title == "" || input.Content == "" {
		http.Error(w, "Title and content are required", http.StatusBadRequest)
		return
	}
	_, err = DB.Exec(`INSERT INTO posts (user_id, title, content, categories) VALUES (?, ?, ?, ?)`,
		userID, input.Title, input.Content, trimAndNormalize(input.Categories))
	if err != nil {
		http.Error(w, "Failed to create post", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func GetPostsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, _, err := GetUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	rows, err := DB.Query(`
		SELECT posts.id, posts.user_id, users.username, posts.title, posts.content,
		       posts.categories, posts.created_at,
		       COALESCE(SUM(CASE WHEN post_reactions.value = 1 THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN post_reactions.value = -1 THEN 1 ELSE 0 END), 0),
		       COALESCE((SELECT value FROM post_reactions WHERE post_id = posts.id AND user_id = ?), 0)
		FROM posts
		JOIN users ON posts.user_id = users.id
		LEFT JOIN post_reactions ON post_reactions.post_id = posts.id
		GROUP BY posts.id
		ORDER BY posts.created_at DESC`, userID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	posts := []Post{}
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.UserID, &post.Author, &post.Title, &post.Content,
			&post.Categories, &post.CreatedAt, &post.Likes, &post.Dislikes, &post.UserReaction); err == nil {
			posts = append(posts, post)
		}
	}
	writeJSON(w, http.StatusOK, posts)
}

type DeletePostInput struct {
	PostID int `json:"post_id"`
}

func DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, _, err := GetUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	input, err := decodeDeletePostInput(r)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	var ownerID int
	err = DB.QueryRow(`SELECT user_id FROM posts WHERE id = ?`, input.PostID).Scan(&ownerID)
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	if ownerID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if err := deletePost(input.PostID); err != nil {
		http.Error(w, "Failed to delete post", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func decodeDeletePostInput(r *http.Request) (DeletePostInput, error) {
	var input DeletePostInput
	if r.Method == http.MethodDelete {
		err := json.NewDecoder(r.Body).Decode(&input)
		return input, err
	}
	input.PostID, _ = strconv.Atoi(r.URL.Query().Get("post_id"))
	if input.PostID <= 0 {
		return input, errInvalidInput
	}
	return input, nil
}

func deletePost(postID int) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM post_reactions WHERE post_id = ?`, postID); err == nil {
		_, err = tx.Exec(`DELETE FROM comments WHERE post_id = ?`, postID)
	}
	if err == nil {
		_, err = tx.Exec(`DELETE FROM posts WHERE id = ?`, postID)
	}
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
