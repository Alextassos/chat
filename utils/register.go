package utils

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type RegisterInput struct {
	Nickname  string `json:"nickname"`
	Age       int    `json:"age"`
	Gender    string `json:"gender"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if trimAndNormalize(input.Nickname) == "" || trimAndNormalize(input.Email) == "" ||
		input.Password == "" || trimAndNormalize(input.FirstName) == "" ||
		trimAndNormalize(input.LastName) == "" || trimAndNormalize(input.Gender) == "" || input.Age <= 0 {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}
	if input.Age < 13 || input.Age > 120 {
		http.Error(w, "Age must be between 13 and 120", http.StatusBadRequest)
		return
	}
	if len(input.Password) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	_, err = DB.Exec(`
		INSERT INTO users (username, password, email, age, first_name, last_name, gender)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		input.Nickname, string(hashedPassword), input.Email, input.Age,
		input.FirstName, input.LastName, input.Gender)
	if err != nil {
		http.Error(w, "Nickname or email already exists", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
