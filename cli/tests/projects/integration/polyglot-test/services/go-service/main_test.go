package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIntegrationHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", resp["status"])
	}
}

func TestIntegrationCalculateAdd(t *testing.T) {
	body := CalculateRequest{
		Operation: "add",
		A:         5,
		B:         3,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/calculate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	calculateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp CalculateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Result != 8 {
		t.Errorf("Expected result 8, got %v", resp.Result)
	}
}

func TestIntegrationCalculateSubtract(t *testing.T) {
	body := CalculateRequest{
		Operation: "subtract",
		A:         10,
		B:         4,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/calculate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	calculateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp CalculateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Result != 6 {
		t.Errorf("Expected result 6, got %v", resp.Result)
	}
}

func TestIntegrationCalculateMultiply(t *testing.T) {
	body := CalculateRequest{
		Operation: "multiply",
		A:         3,
		B:         7,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/calculate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	calculateHandler(w, req)

	var resp CalculateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Result != 21 {
		t.Errorf("Expected result 21, got %v", resp.Result)
	}
}

func TestIntegrationCalculateDivide(t *testing.T) {
	body := CalculateRequest{
		Operation: "divide",
		A:         20,
		B:         4,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/calculate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	calculateHandler(w, req)

	var resp CalculateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Result != 5 {
		t.Errorf("Expected result 5, got %v", resp.Result)
	}
}

func TestIntegrationCalculateDivideByZero(t *testing.T) {
	body := CalculateRequest{
		Operation: "divide",
		A:         10,
		B:         0,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/calculate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	calculateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var resp CalculateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error == "" {
		t.Error("Expected error message for division by zero")
	}
}

func TestIntegrationCalculateUnknownOperation(t *testing.T) {
	body := CalculateRequest{
		Operation: "unknown",
		A:         1,
		B:         2,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/calculate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	calculateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestIntegrationCalculateMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/calculate", nil)
	w := httptest.NewRecorder()

	calculateHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestIntegrationCalculateInvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/calculate", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	calculateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
