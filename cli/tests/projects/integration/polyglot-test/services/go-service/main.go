// Package main is the entry point for the go-service.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jongio/azd-app/tests/polyglot-test/go-service/calculator"
)

// CalculateRequest represents a calculation request.
type CalculateRequest struct {
	Operation string  `json:"operation"`
	A         float64 `json:"a"`
	B         float64 `json:"b"`
}

// CalculateResponse represents a calculation response.
type CalculateResponse struct {
	Result float64 `json:"result,omitempty"`
	Error  string  `json:"error,omitempty"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func calculateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CalculateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var result float64
	var err error

	switch req.Operation {
	case "add":
		result = calculator.Add(req.A, req.B)
	case "subtract":
		result = calculator.Subtract(req.A, req.B)
	case "multiply":
		result = calculator.Multiply(req.A, req.B)
	case "divide":
		result, err = calculator.Divide(req.A, req.B)
	default:
		http.Error(w, fmt.Sprintf("Unknown operation: %s", req.Operation), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp := CalculateResponse{}
	if err != nil {
		resp.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
	} else {
		resp.Result = result
	}
	json.NewEncoder(w).Encode(resp)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/calculate", calculateHandler)

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
