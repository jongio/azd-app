package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// writeJSON writes a JSON response with proper error handling.
func writeJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return fmt.Errorf("failed to encode JSON response: %w", err)
	}
	return nil
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, statusCode int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"error": message,
	}
	if err != nil {
		response["details"] = err.Error()
	}

	_ = json.NewEncoder(w).Encode(response)
}

// writeWebSocketJSON safely writes JSON to a WebSocket connection with mutex protection.
func (c *clientConn) writeWebSocketJSON(data interface{}) error {
	if c.client == nil {
		return fmt.Errorf("WebSocket client not initialized")
	}
	return c.client.writeJSON(data)
}
