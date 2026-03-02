package dashboard

import (
	"fmt"
	"log"
	"net/http"
)

// Constants for error messages - these don't duplicate existing constants
const (
	errInvalidJSON            = "Invalid JSON"
	errRequestBodyTooLarge    = "Request body too large"
	errFailedToReadBody       = "Failed to read request body"
	errServiceRequired        = "service parameter required"
	errModeRequired           = "mode parameter required"
	errQueryRequired          = "query parameter required"
	errTextRequired           = "Text is required"
	errInvalidIndex           = "Invalid index"
	errIndexRequired          = "Index required"
	errIndexOutOfRange        = "Index out of range"
	errNoClassificationsFound = "No classifications found"
)

// MethodGuard is a middleware that validates HTTP methods for a handler.
// It returns a wrapped handler that only allows the specified methods.
// If the request method doesn't match any allowed methods, it returns 405 Method Not Allowed.
//
// Example usage:
//
//	http.HandleFunc("/api/resource", MethodGuard(handler, http.MethodGet, http.MethodPost))
func MethodGuard(handler http.HandlerFunc, allowedMethods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, method := range allowedMethods {
			if r.Method == method {
				handler(w, r)
				return
			}
		}
		http.Error(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}

// HandleLoadError writes a standardized error response for azure.yaml load failures.
// This eliminates duplicate error handling code across handlers.
func HandleLoadError(w http.ResponseWriter, err error) {
	writeJSONError(w, http.StatusInternalServerError, errLoadAzureYamlFailed, err)
}

// HandleSaveError writes a standardized error response for azure.yaml save failures.
// This eliminates duplicate error handling code across handlers.
func HandleSaveError(w http.ResponseWriter, err error) {
	writeJSONError(w, http.StatusInternalServerError, errSaveAzureYamlFailed, err)
}

// ErrorResponse represents a structured error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// BadRequest writes a 400 Bad Request error response with a custom message.
func BadRequest(w http.ResponseWriter, message string, err error) {
	writeJSONError(w, http.StatusBadRequest, message, err)
}

// NotFound writes a 404 Not Found error response with a custom message.
func NotFound(w http.ResponseWriter, message string) {
	writeJSONError(w, http.StatusNotFound, message, nil)
}

// InternalError writes a 500 Internal Server Error response with a custom message.
func InternalError(w http.ResponseWriter, message string, err error) {
	writeJSONError(w, http.StatusInternalServerError, message, err)
}

// Note: readLimitedBody, decodeJSON, and parseIntParam are already defined in azure_logs_conversion.go
// so we don't redefine them here to avoid duplication

// RequireQueryParam checks if a required query parameter is present and non-empty.
// If missing, writes a 400 Bad Request error and returns false.
// If present, returns true and the caller should continue processing.
func RequireQueryParam(w http.ResponseWriter, r *http.Request, paramName string) (string, bool) {
	value := r.URL.Query().Get(paramName)
	if value == "" {
		BadRequest(w, fmt.Sprintf("%s parameter required", paramName), nil)
		return "", false
	}
	return value, true
}

// WriteJSONSuccess writes a successful JSON response with 200 OK status.
func WriteJSONSuccess(w http.ResponseWriter, data interface{}) {
	if err := writeJSON(w, data); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}

// WriteJSONCreated writes a successful JSON response with 201 Created status.
func WriteJSONCreated(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusCreated)
	if err := writeJSON(w, data); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}

// WriteNoContent writes a 204 No Content response.
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// securityHeaders is middleware that adds security headers to all HTTP responses.
// These headers provide defense-in-depth against common web attacks (CWE-693).
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws://localhost:* wss://localhost:*; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// ReadJSONBody reads and decodes a JSON request body with size limits.
// Returns an error if the body is too large, cannot be read, or contains invalid JSON.
func ReadJSONBody(w http.ResponseWriter, r *http.Request, target interface{}, maxSize int64) bool {
	body, err := readLimitedBody(r, maxSize)
	if err != nil {
		BadRequest(w, errFailedToReadBody, err)
		return false
	}

	if err := decodeJSON(body, target); err != nil {
		BadRequest(w, errInvalidJSON, err)
		return false
	}

	return true
}
