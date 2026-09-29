package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// RubyGems looks up package metadata from the rubygems.org API.
// https://guides.rubygems.org/rubygems-org-api/
type RubyGems struct {
	BaseURL string
	Client  *http.Client
}

func NewRubyGems() *RubyGems {
	return &RubyGems{BaseURL: "https://rubygems.org", Client: newClient()}
}

func (r *RubyGems) Name() string { return "rubygems" }

type rubygemsVersion struct {
	CreatedAt string `json:"created_at"`
}

func (r *RubyGems) Lookup(ctx context.Context, name string) (Info, error) {
	reqURL := r.BaseURL + "/api/v1/gems/" + url.PathEscape(name) + ".json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return Info{}, err
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return Info{}, &UnreachableError{Registry: r.Name(), Err: err}
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Info{Exists: false}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return Info{}, &UnreachableError{Registry: r.Name(), Err: httpStatusErr(resp.StatusCode)}
	}

	info := Info{Exists: true}

	versionsURL := r.BaseURL + "/api/v1/versions/" + url.PathEscape(name) + ".json"
	vreq, err := http.NewRequestWithContext(ctx, http.MethodGet, versionsURL, nil)
	if err != nil {
		return info, nil
	}
	vresp, err := r.Client.Do(vreq)
	if err != nil {
		// The gem exists but we could not date it. Do not fail the whole
		// lookup over a second, non-essential call.
		return info, nil
	}
	defer vresp.Body.Close()
	if vresp.StatusCode != http.StatusOK {
		return info, nil
	}
	var versions []rubygemsVersion
	if err := json.NewDecoder(vresp.Body).Decode(&versions); err != nil {
		return info, nil
	}
	var earliest time.Time
	for _, v := range versions {
		if v.CreatedAt == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, v.CreatedAt)
		if err != nil {
			continue
		}
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
	}
	if !earliest.IsZero() {
		info.CreatedAt = earliest
		info.HasCreatedAt = true
	}
	return info, nil
}
