package utils

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	ID         int    `json:"id"`
	SenderID   int    `json:"sender_id"`
	ReceiverID int    `json:"receiver_id"`
	Author     string `json:"author"`
	Content    string `json:"content"`
	CreatedAt  string `json:"created_at"`
}

type ChatUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
	LastAt   string `json:"last_at,omitempty"`
	Unread   int    `json:"unread"`
}

type wsClient struct {
	userID int
	conn   *websocket.Conn
	send   chan []byte
}

var chatHub = struct {
	sync.Mutex
	clients map[int]map[*wsClient]bool
}{
	clients: make(map[int]map[*wsClient]bool),
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
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
		SELECT users.id, users.username,
		       COALESCE((
		           SELECT created_at FROM messages
		           WHERE (sender_id = users.id AND receiver_id = ?)
		      OR (sender_id = ? AND receiver_id = users.id)
		      ORDER BY id DESC LIMIT 1
		       ), '') AS last_at,
		       COALESCE((
		           SELECT COUNT(*)
		           FROM messages
		           WHERE messages.sender_id = users.id
		             AND messages.receiver_id = ?
		             AND messages.id > COALESCE((
		                 SELECT last_message_id
		                 FROM message_read_states
		                 WHERE user_id = ? AND contact_id = users.id
		             ), 0)
		       ), 0) AS unread
		FROM users
		WHERE users.id != ?
		ORDER BY CASE WHEN last_at = '' THEN 1 ELSE 0 END,
		         last_at DESC, users.username ASC`,
		userID, userID, userID, userID, userID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []ChatUser{}
	for rows.Next() {
		var user ChatUser
		if err := rows.Scan(&user.ID, &user.Username, &user.LastAt, &user.Unread); err == nil {
			user.Online = isUserOnline(user.ID)
			users = append(users, user)
		}
	}
	writeJSON(w, http.StatusOK, users)
}

func GetMessagesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, _, err := GetUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	otherID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil || otherID <= 0 {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}
	limit := 10
	beforeID := 0
	if value, err := strconv.Atoi(r.URL.Query().Get("before_id")); err == nil {
		beforeID = value
	}

	query := `
		SELECT messages.id, messages.sender_id, messages.receiver_id,
		       users.username, messages.content, messages.created_at
		FROM messages JOIN users ON users.id = messages.sender_id
		WHERE ((messages.sender_id = ? AND messages.receiver_id = ?)
		    OR (messages.sender_id = ? AND messages.receiver_id = ?))`
	args := []any{userID, otherID, otherID, userID}
	if beforeID > 0 {
		query += ` AND messages.id < ?`
		args = append(args, beforeID)
	}
	query += ` ORDER BY messages.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := DB.Query(query, args...)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var message Message
		if err := rows.Scan(&message.ID, &message.SenderID, &message.ReceiverID,
			&message.Author, &message.Content, &message.CreatedAt); err == nil {
			messages = append(messages, message)
		}
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	writeJSON(w, http.StatusOK, messages)
}

type CreateMessageInput struct {
	ReceiverID int    `json:"receiver_id"`
	Content    string `json:"content"`
}

func CreateMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	senderID, senderName, err := GetUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var input CreateMessageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	input.Content = trimAndNormalize(input.Content)
	if input.ReceiverID <= 0 || input.Content == "" {
		http.Error(w, "Receiver and content are required", http.StatusBadRequest)
		return
	}
	var receiverExists int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ?`, input.ReceiverID).Scan(&receiverExists); err != nil || receiverExists == 0 {
		http.Error(w, "Receiver not found", http.StatusNotFound)
		return
	}

	result, err := DB.Exec(
		`INSERT INTO messages (sender_id, receiver_id, content) VALUES (?, ?, ?)`,
		senderID, input.ReceiverID, input.Content,
	)
	if err != nil {
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}
	messageID, _ := result.LastInsertId()
	message := Message{
		ID: int(messageID), SenderID: senderID, ReceiverID: input.ReceiverID,
		Author: senderName, Content: input.Content, CreatedAt: time.Now().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(map[string]any{"type": "message", "message": message})
	broadcastToUser(input.ReceiverID, payload)
	broadcastToUser(senderID, payload)
	broadcastPresence()
	writeJSON(w, http.StatusCreated, message)
}

type MarkMessagesReadInput struct {
	ContactID int `json:"contact_id"`
}

func MarkMessagesReadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, _, err := GetUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var input MarkMessagesReadInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ContactID <= 0 {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	var lastMessageID int
	err = DB.QueryRow(`
		SELECT COALESCE(MAX(id), 0)
		FROM messages
		WHERE sender_id = ? AND receiver_id = ?`,
		input.ContactID, userID,
	).Scan(&lastMessageID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	_, err = DB.Exec(`
		INSERT INTO message_read_states (user_id, contact_id, last_message_id)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, contact_id)
		DO UPDATE SET last_message_id = excluded.last_message_id`,
		userID, input.ContactID, lastMessageID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	userID, _, err := GetUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &wsClient{userID: userID, conn: conn, send: make(chan []byte, 32)}
	registerClient(client)
	go writeWebSocket(client)
	readWebSocket(client)
}

func registerClient(client *wsClient) {
	chatHub.Lock()
	if chatHub.clients[client.userID] == nil {
		chatHub.clients[client.userID] = make(map[*wsClient]bool)
	}
	chatHub.clients[client.userID][client] = true
	chatHub.Unlock()
	broadcastPresence()
}

func unregisterClient(client *wsClient) {
	chatHub.Lock()
	delete(chatHub.clients[client.userID], client)
	if len(chatHub.clients[client.userID]) == 0 {
		delete(chatHub.clients, client.userID)
	}
	chatHub.Unlock()
	client.conn.Close()
	broadcastPresence()
}

func readWebSocket(client *wsClient) {
	defer unregisterClient(client)
	for {
		_, payload, err := client.conn.ReadMessage()
		if err != nil {
			return
		}
		var event struct {
			Type       string `json:"type"`
			ReceiverID int    `json:"receiver_id"`
			Typing     bool   `json:"typing"`
		}
		if err := json.Unmarshal(payload, &event); err != nil {
			continue
		}
		if event.Type == "typing" && event.ReceiverID > 0 {
			packet, _ := json.Marshal(map[string]any{
				"type":        "typing",
				"sender_id":   client.userID,
				"receiver_id": event.ReceiverID,
				"typing":      event.Typing,
			})
			broadcastToUser(event.ReceiverID, packet)
		}
	}
}

func writeWebSocket(client *wsClient) {
	defer client.conn.Close()
	for payload := range client.send {
		client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := client.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return
		}
	}
}

func isUserOnline(userID int) bool {
	chatHub.Lock()
	defer chatHub.Unlock()
	return len(chatHub.clients[userID]) > 0
}

func broadcastToUser(userID int, payload []byte) {
	chatHub.Lock()
	clients := make([]*wsClient, 0)
	for client := range chatHub.clients[userID] {
		clients = append(clients, client)
	}
	chatHub.Unlock()
	for _, client := range clients {
		select {
		case client.send <- payload:
		default:
		}
	}
}

func broadcastPresence() {
	rows, err := DB.Query(`SELECT id, username FROM users`)
	if err != nil {
		return
	}
	defer rows.Close()
	users := []ChatUser{}
	for rows.Next() {
		var user ChatUser
		if err := rows.Scan(&user.ID, &user.Username); err == nil {
			user.Online = isUserOnline(user.ID)
			users = append(users, user)
		}
	}
	payload, _ := json.Marshal(map[string]any{"type": "presence", "users": users})
	chatHub.Lock()
	clients := make([]*wsClient, 0)
	for _, userClients := range chatHub.clients {
		for client := range userClients {
			clients = append(clients, client)
		}
	}
	chatHub.Unlock()
	for _, client := range clients {
		select {
		case client.send <- payload:
		default:
		}
	}
}
