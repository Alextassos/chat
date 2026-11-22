package utils

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type PostReactionInput struct {
	PostID int `json:"post_id"`
	Value  int `json:"value"`
}

func PostReactionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, _, err := GetUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var input PostReactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if input.PostID <= 0 || (input.Value != 1 && input.Value != -1) {
		http.Error(w, "Invalid reaction", http.StatusBadRequest)
		return
	}

	var existing int
	err = DB.QueryRow(
		`SELECT value FROM post_reactions WHERE post_id = ? AND user_id = ?`,
		input.PostID, userID,
	).Scan(&existing)

	switch {
	case err == nil && existing == input.Value:
		_, err = DB.Exec(`DELETE FROM post_reactions WHERE post_id = ? AND user_id = ?`, input.PostID, userID)
	case err == nil:
		_, err = DB.Exec(`UPDATE post_reactions SET value = ? WHERE post_id = ? AND user_id = ?`,
			input.Value, input.PostID, userID)
	case err == sql.ErrNoRows:
		_, err = DB.Exec(`INSERT INTO post_reactions (post_id, user_id, value) VALUES (?, ?, ?)`,
			input.PostID, userID, input.Value)
	}
	if err != nil {
		http.Error(w, "Failed to react", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
