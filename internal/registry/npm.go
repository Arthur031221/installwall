package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NPM looks up package metadata from the public npm registry.
// https://github.com/npm/registry/blob/main/docs/REGISTRY-API.md
type NPM struct {
	BaseURL string
	Client  *http.Client
}

func NewNPM() *NPM {
	return &NPM{BaseURL: "https://registry.npmjs.org", Client: newClient()}
}

func (r *NPM) Name() string { return "npm" }

type npmDoc struct {
	Time map[string]string `json:"time"`
}

func (r *NPM) Lookup(ctx context.Context, name string) (Info, error) {
	// Scoped packages (@scope/name) need the slash escaped as %2f.
	escaped := strings.Replace(url.PathEscape(name), "%2F", "%2f", 1)
	reqURL := r.BaseURL + "/" + escaped
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return Info{}, err
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return Info{}, &UnreachableError{Registry: r.Name(), Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Info{Exists: false}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return Info{}, &UnreachableError{Registry: r.Name(), Err: httpStatusErr(resp.StatusCode)}
	}
	var doc npmDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return Info{}, &UnreachableError{Registry: r.Name(), Err: err}
	}
	info := Info{Exists: true}
	if created, ok := doc.Time["created"]; ok {
		if t, err := time.Parse(time.RFC3339, created); err == nil {
			info.CreatedAt = t
			info.HasCreatedAt = true
		}
	}
	return info, nil
}
