package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// DecodeBody reads and JSON-decodes the request body into dst.
// Unknown fields are rejected. Returns false and writes a 400 response on failure.
func DecodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return false
	}
	return true
}

// PathParam extracts a named path parameter from the request.
// Wraps r.PathValue() (Go 1.22+).
func PathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}

// QueryInt reads an integer query parameter, returning the default if absent or unparseable.
func QueryInt(r *http.Request, name string, defaultVal int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return v
}

// QueryString reads a string query parameter, returning the default if absent.
func QueryString(r *http.Request, name string, defaultVal string) string {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultVal
	}
	return raw
}
