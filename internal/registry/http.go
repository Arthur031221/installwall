package registry

import (
	"net/http"
	"time"
)

// newClient returns an HTTP client with a short, fixed timeout. Registry
// lookups run on every install; they must fail fast so a slow registry
// degrades to a warning instead of hanging the agent's install command.
func newClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}
