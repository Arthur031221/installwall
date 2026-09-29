package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// Crates looks up package metadata from the crates.io API. crates.io asks
// every script client to send a descriptive User-Agent with contact info,
// see https://crates.io/policies#crawlers.
type Crates struct {
	BaseURL   string
	Client    *http.Client
	UserAgent string
}

func NewCrates() *Crates {
	return &Crates{
		BaseURL:   "https://crates.io",
		Client:    newClient(),
		UserAgent: "installwall (https://github.com/Arthur031221/installwall)",
	}
}

func (r *Crates) Name() string { return "crates" }

type cratesDoc struct {
	Crate struct {
		CreatedAt string `json:"created_at"`
	} `json:"crate"`
}

func (r *Crates) Lookup(ctx context.Context, name string) (Info, error) {
	reqURL := r.BaseURL + "/api/v1/crates/" + url.PathEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return Info{}, err
	}
	if r.UserAgent != "" {
		req.Header.Set("User-Agent", r.UserAgent)
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
	var doc cratesDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return Info{}, &UnreachableError{Registry: r.Name(), Err: err}
	}
	info := Info{Exists: true}
	if doc.Crate.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, doc.Crate.CreatedAt); err == nil {
			info.CreatedAt = t
			info.HasCreatedAt = true
		}
	}
	return info, nil
}
