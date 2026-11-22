package utils

import (
	"encoding/json"
	"net/http"
	"time"
)

func GetUserFromRequest(r *http.Request) (int, string, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return 0, "", err
	}
	var userID int
	var username string
	query := `SELECT users.id, users.username FROM sessions 
          JOIN users ON sessions.user_id = users.id 
          WHERE sessions.token = ? AND sessions.expires_at > ?`
	err = DB.QueryRow(query, cookie.Value, time.Now()).Scan(&userID, &username)
	return userID, username, err

}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	userID, username, err := GetUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":       userID,
		"username": username,
	})

}
