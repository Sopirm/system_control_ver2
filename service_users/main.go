package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()

	// Маршруты для сервиса пользователей
	router.HandleFunc("/v1/users/register", registerUser).Methods("POST")
	router.HandleFunc("/v1/users/login", loginUser).Methods("POST")
	router.HandleFunc("/v1/users/profile", getUserProfile).Methods("GET")
	router.HandleFunc("/v1/users/profile", updateUserProfile).Methods("PUT")
	router.HandleFunc("/v1/users", listUsers).Methods("GET")

	fmt.Println("Service Users запущен на порту :8081")
	log.Fatal(http.ListenAndServe(":8081", router))
}

// registerUser обрабатывает регистрацию нового пользователя
func registerUser(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "Register User Endpoint"})
}

// loginUser обрабатывает вход пользователя и выдачу JWT
func loginUser(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "Login User Endpoint"})
}

// getUserProfile возвращает профиль текущего пользователя
func getUserProfile(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "Get User Profile Endpoint"})
}

// updateUserProfile обновляет профиль текущего пользователя
func updateUserProfile(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "Update User Profile Endpoint"})
}

// listUsers возвращает список пользователей (для администраторов)
func listUsers(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "List Users Endpoint"})
}
