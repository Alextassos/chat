package main

import (
	"chat/utils"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	databasePath := os.Getenv("DATABASE_PATH")
	if databasePath == "" {
		databasePath = "./forum.db"
	}

	err := utils.InitDB(databasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer utils.DB.Close()
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/api/register", utils.RegisterHandler)
	http.HandleFunc("/api/login", utils.LoginHandler)
	http.HandleFunc("/api/logout", utils.LogoutHandler)
	http.HandleFunc("/api/me", utils.MeHandler)
	http.HandleFunc("/api/posts/create", utils.CreatePostHandler)
	http.HandleFunc("/api/posts/delete", utils.DeletePostHandler)
	http.HandleFunc("/api/posts/reaction", utils.PostReactionHandler)
	http.HandleFunc("/api/comments/create", utils.CreateCommentHandler)
	http.HandleFunc("/api/posts", utils.GetPostsHandler)
	http.HandleFunc("/api/comments", utils.GetCommentsHandler)
	http.HandleFunc("/api/users", utils.GetUsersHandler)
	http.HandleFunc("/api/messages", utils.GetMessagesHandler)
	http.HandleFunc("/api/messages/create", utils.CreateMessageHandler)
	http.HandleFunc("/api/messages/read", utils.MarkMessagesReadHandler)
	http.HandleFunc("/ws", utils.WebSocketHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./templates/index.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	address := ":" + port
	fmt.Printf("Server started at %s\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
