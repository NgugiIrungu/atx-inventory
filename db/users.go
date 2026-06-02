package db

import (
	"database/sql"
	"errors"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// User — mirrors the users table
type User struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// CreateUser — registers a new user with a hashed password
func CreateUser(username, email, password, role string) (*User, error) {
	// Hash the password
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	result, err := DB.Exec(`
		INSERT INTO users (username, email, password, role)
		VALUES (?, ?, ?, ?)
	`, username, email, string(hashed), role)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	log.Printf("✓ CreateUser: %s [%s]", username, role)
	return &User{Id: int(id), Username: username, Email: email, Role: role}, nil
}

// AuthenticateUser — checks username and password, returns user if valid
func AuthenticateUser(username, password string) (*User, error) {
	var hashed string
	user := &User{}

	err := DB.QueryRow(`
		SELECT id, username, email, role, password
		FROM users WHERE username = ?
	`, username).Scan(&user.Id, &user.Username, &user.Email, &user.Role, &hashed)

	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	// Compare the password with the hash
	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)); err != nil {
		return nil, errors.New("incorrect password")
	}

	return user, nil
}

// GetUserByID — returns a user by their ID
func GetUserByID(id int) (*User, error) {
	user := &User{}
	err := DB.QueryRow(`
		SELECT id, username, email, role FROM users WHERE id = ?
	`, id).Scan(&user.Id, &user.Username, &user.Email, &user.Role)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateAdminPassword — sets a proper hashed password for the default admin
func UpdateAdminPassword() {
	hashed, _ := bcrypt.GenerateFromPassword([]byte("atx@admin2024"), bcrypt.DefaultCost)
	DB.Exec("UPDATE users SET password = ? WHERE username = 'admin'", string(hashed))
	log.Println("✓ Admin password set to: atx@admin2024")
}