
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// User represents a user in our system
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// In-memory storage
var (
	users   []User
	nextID  int = 1
	usersMu sync.RWMutex // Mutex for thread safety
)

// Helper function to find user by ID
func findUserByID(id int) (*User, int) {
	for i, user := range users {
		if user.ID == id {
			return &user, i
		}
	}
	return nil, -1
}

// Helper function to check if email exists
func emailExists(email string, excludeID int) bool {
	for _, user := range users {
		if user.Email == email && user.ID != excludeID {
			return true
		}
	}
	return false
}

// CREATE - Add a new user
func CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if user.Name == "" || user.Email == "" || user.Age <= 0 {
		http.Error(w, "Name, email, and age are required", http.StatusBadRequest)
		return
	}

	usersMu.Lock()
	defer usersMu.Unlock()

	// Check if email already exists
	if emailExists(user.Email, -1) {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	// Set user fields
	user.ID = nextID
	nextID++
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Add to users slice
	users = append(users, user)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// READ - Get all users
func GetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	usersMu.RLock()
	defer usersMu.RUnlock()

	// Return empty array if no users
	if len(users) == 0 {
		json.NewEncoder(w).Encode([]User{})
		return
	}

	json.NewEncoder(w).Encode(users)
}

// READ - Get user by ID
func GetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	usersMu.RLock()
	defer usersMu.RUnlock()

	user, _ := findUserByID(id)
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(*user)
}

// UPDATE - Update user by ID
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var updatedUser User
	err = json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if updatedUser.Name == "" || updatedUser.Email == "" || updatedUser.Age <= 0 {
		http.Error(w, "Name, email, and age are required", http.StatusBadRequest)
		return
	}

	usersMu.Lock()
	defer usersMu.Unlock()

	user, index := findUserByID(id)
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if email already exists for another user
	if emailExists(updatedUser.Email, id) {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	// Update user fields
	users[index].Name = updatedUser.Name
	users[index].Email = updatedUser.Email
	users[index].Age = updatedUser.Age
	users[index].UpdatedAt = time.Now()

	json.NewEncoder(w).Encode(users[index])
}

// DELETE - Delete user by ID
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	usersMu.Lock()
	defer usersMu.Unlock()

	_, index := findUserByID(id)
	if index == -1 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Remove user from slice
	users = append(users[:index], users[index+1:]...)

	response := map[string]string{"message": "User deleted successfully"}
	json.NewEncoder(w).Encode(response)
}

// GET - Get users count
func GetUsersCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	usersMu.RLock()
	defer usersMu.RUnlock()

	response := map[string]int{"count": len(users)}
	json.NewEncoder(w).Encode(response)
}

// DELETE - Clear all users (bonus endpoint)
func ClearUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	usersMu.Lock()
	defer usersMu.Unlock()

	users = []User{}
	nextID = 1

	response := map[string]string{"message": "All users cleared successfully"}
	json.NewEncoder(w).Encode(response)
}

// setupRoutes configures all the routes
func setupRoutes() *mux.Router {
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/users", CreateUser).Methods("POST")
	api.HandleFunc("/users", GetUsers).Methods("GET")
	api.HandleFunc("/users/count", GetUsersCount).Methods("GET")
	api.HandleFunc("/users/clear", ClearUsers).Methods("DELETE")
	api.HandleFunc("/users/{id}", GetUser).Methods("GET")
	api.HandleFunc("/users/{id}", UpdateUser).Methods("PUT")
	api.HandleFunc("/users/{id}", DeleteUser).Methods("DELETE")

	return router
}

// Initialize with some sample data
func initSampleData() {
	usersMu.Lock()
	defer usersMu.Unlock()

	now := time.Now()
	sampleUsers := []User{
		{
			ID:        1,
			Name:      "John Doe",
			Email:     "john@example.com",
			Age:       30,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        2,
			Name:      "Jane Smith",
			Email:     "jane@example.com",
			Age:       25,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	users = append(users, sampleUsers...)
	nextID = 3
}

func main() {
	// Initialize with sample data
	initSampleData()

	// Setup routes
	router := setupRoutes()

	// Start server
	port := ":8080"
	fmt.Printf("Server starting on port %s\n", port)
	fmt.Println("\nAPI Endpoints:")
	fmt.Println("POST   /api/v1/users        - Create a new user")
	fmt.Println("GET    /api/v1/users        - Get all users")
	fmt.Println("GET    /api/v1/users/count  - Get users count")
	fmt.Println("GET    /api/v1/users/{id}   - Get user by ID")
	fmt.Println("PUT    /api/v1/users/{id}   - Update user by ID")
	fmt.Println("DELETE /api/v1/users/{id}   - Delete user by ID")
	fmt.Println("DELETE /api/v1/users/clear  - Clear all users")
	fmt.Println("\nSample users are pre-loaded!")

	log.Fatal(http.ListenAndServe(port, router))
}