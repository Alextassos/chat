package utils

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(dataSourceName string) error {
	var err error
	DB, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return err
	}

	userQuery := `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE,
	age INTEGER,
	first_name TEXT,
	last_name TEXT,
	gender TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`
	_, err = DB.Exec(userQuery)
	if err != nil {
		return err
	}
	sessionQuery := `
CREATE TABLE IF NOT EXISTS sessions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	token TEXT NOT NULL UNIQUE,
	expires_at DATETIME NOT NULL
);`

	_, err = DB.Exec(sessionQuery)
	if err != nil {
		return err
	}
	postsQuery := `
CREATE TABLE IF NOT EXISTS posts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	title TEXT NOT NULL,
	content TEXT NOT NULL,
	categories TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`
	_, err = DB.Exec(postsQuery)
	if err != nil {
		return err
	}
	commentsQuery := `
CREATE TABLE IF NOT EXISTS comments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	post_id INTEGER NOT NULL,
	content TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`
	_, err = DB.Exec(commentsQuery)
	if err != nil {
		return err
	}
	messagesQuery := `
CREATE TABLE IF NOT EXISTS messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,		
sender_id INTEGER NOT NULL,
receiver_id INTEGER NOT NULL,
content TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`
	_, err = DB.Exec(messagesQuery)
	if err != nil {
		return err
	}
	reactionsQuery := `
CREATE TABLE IF NOT EXISTS post_reactions (
	post_id INTEGER NOT NULL,
	user_id INTEGER NOT NULL,
	value INTEGER NOT NULL CHECK (value IN (-1, 1)),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(post_id, user_id)
);`
	_, err = DB.Exec(reactionsQuery)
	if err != nil {
		return err
	}
	readStatesQuery := `
CREATE TABLE IF NOT EXISTS message_read_states (
	user_id INTEGER NOT NULL,
	contact_id INTEGER NOT NULL,
	last_message_id INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (user_id, contact_id)
);`
	_, err = DB.Exec(readStatesQuery)
	if err != nil {
		return err
	}
	DB.SetMaxOpenConns(1)

	return nil

}
