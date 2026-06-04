package sse

import (
	"fmt"
	"net/http"
	"time"
)

// DisableWriteTimeout clears the per-request write deadline set by http.Server.
// Long-lived SSE responses must outlive the normal API WriteTimeout.
func DisableWriteTimeout(w http.ResponseWriter) error {
	return http.NewResponseController(w).SetWriteDeadline(time.Time{})
}

// WriteComment emits a standards-compliant SSE comment frame.
func WriteComment(w http.ResponseWriter, value string) error {
	_, err := fmt.Fprintf(w, ": %s\n\n", value)
	return err
}
