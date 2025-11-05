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

	// Маршруты для сервиса заказов
	router.HandleFunc("/v1/orders", createOrder).Methods("POST")
	router.HandleFunc("/v1/orders/{id}", getOrder).Methods("GET")
	router.HandleFunc("/v1/orders", listOrders).Methods("GET")
	router.HandleFunc("/v1/orders/{id}/status", updateOrderStatus).Methods("PUT")
	router.HandleFunc("/v1/orders/{id}/cancel", cancelOrder).Methods("PUT")

	fmt.Println("Service Orders запущен на порту :8082")
	log.Fatal(http.ListenAndServe(":8082", router))
}

// createOrder обрабатывает создание нового заказа
func createOrder(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "Create Order Endpoint"})
}

// getOrder возвращает заказ по идентификатору
func getOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]
	json.NewEncoder(w).Encode(map[string]string{"message": "Get Order Endpoint", "order_id": orderID})
}

// listOrders возвращает список заказов текущего пользователя
func listOrders(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "List Orders Endpoint"})
}

// updateOrderStatus обновляет статус заказа
func updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]
	json.NewEncoder(w).Encode(map[string]string{"message": "Update Order Status Endpoint", "order_id": orderID})
}

// cancelOrder отменяет заказ
func cancelOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]
	json.NewEncoder(w).Encode(map[string]string{"message": "Cancel Order Endpoint", "order_id": orderID})
}
