// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// convertAzureLogLevel converts azure.LogLevel to service.LogLevel.
func convertAzureLogLevel(azLevel azure.LogLevel) service.LogLevel {
	switch azLevel {
	case azure.LogLevelInfo:
		return service.LogLevelInfo
	case azure.LogLevelWarn:
		return service.LogLevelWarn
	case azure.LogLevelError:
		return service.LogLevelError
	case azure.LogLevelDebug:
		return service.LogLevelDebug
	default:
		return service.LogLevelInfo
	}
}

// parseIntParam parses an integer query parameter.
func parseIntParam(s string) (int, error) {
	var n int
	_, err := parseIntParamWithFormat(s, &n)
	return n, err
}

// parseIntParamWithFormat parses using Sscanf.
func parseIntParamWithFormat(s string, n *int) (int, error) {
	return parseIntFromString(s, n)
}

// parseIntFromString parses an integer from a string.
func parseIntFromString(s string, n *int) (int, error) {
	var count int
	var val int
	// Use simple parsing
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		val = val*10 + int(c-'0')
		count++
	}
	if count > 0 {
		*n = val
		return count, nil
	}
	return 0, errInvalidInt
}

var errInvalidInt = &invalidIntError{}

type invalidIntError struct{}

func (e *invalidIntError) Error() string {
	return "invalid integer"
}

// readLimitedBody reads up to maxSize bytes from the request body.
func readLimitedBody(r *http.Request, maxSize int64) ([]byte, error) {
	return readBodyWithLimit(r.Body, maxSize)
}

type reader interface {
	Read([]byte) (int, error)
}

// readBodyWithLimit reads up to maxSize from a reader.
func readBodyWithLimit(r reader, maxSize int64) ([]byte, error) {
	data := make([]byte, 0, 1024)
	buf := make([]byte, 1024)
	var total int64
	for {
		n, err := r.Read(buf)
		if n > 0 {
			total += int64(n)
			if total > maxSize {
				return nil, errBodyTooLarge
			}
			data = append(data, buf[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
	}
	return data, nil
}

var errBodyTooLarge = &bodyTooLargeError{}

type bodyTooLargeError struct{}

func (e *bodyTooLargeError) Error() string {
	return "request body too large"
}

// decodeJSON unmarshals JSON from bytes.
func decodeJSON(data []byte, v interface{}) error {
	return jsonUnmarshal(data, v)
}

// jsonUnmarshal is a wrapper for JSON unmarshaling.
func jsonUnmarshal(data []byte, v interface{}) error {
	i := 0
	return unmarshalJSONValue(data, &i, v)
}

// unmarshalJSONValue is a simple JSON parser (delegate to encoding/json in practice).
// For production use, this would use encoding/json.Unmarshal
func unmarshalJSONValue(data []byte, _ *int, v interface{}) error {
	// Use standard library
	return parseJSONStandard(data, v)
}

// parseJSONStandard uses standard encoding/json.
func parseJSONStandard(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
