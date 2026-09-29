package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// PyPI looks up package metadata from pypi.org's JSON API.
// https://warehouse.pypa.io/api-reference/json.html
type PyPI struct {
	BaseURL string
	Client  *http.Client
}

func NewPyPI() *PyPI {
	return &PyPI{BaseURL: "https://pypi.org", Client: newClient()}
}

func (r *PyPI) Name() string { return "pypi" }

type pypiFile struct {
	UploadTimeISO string `json:"upload_time_iso_8601"`
}

type pypiDoc struct {
	Releases map[string][]pypiFile `json:"releases"`
}

func (r *PyPI) Lookup(ctx context.Context, name string) (Info, error) {
	reqURL := r.BaseURL + "/pypi/" + url.PathEscape(name) + "/json"
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
	var doc pypiDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return Info{}, &UnreachableError{Registry: r.Name(), Err: err}
	}
	info := Info{Exists: true}
	var earliest time.Time
	for _, files := range doc.Releases {
		for _, f := range files {
			if f.UploadTimeISO == "" {
				continue
			}
			t, err := time.Parse(time.RFC3339, f.UploadTimeISO)
			if err != nil {
				continue
			}
			if earliest.IsZero() || t.Before(earliest) {
				earliest = t
			}
		}
	}
	if !earliest.IsZero() {
		info.CreatedAt = earliest
		info.HasCreatedAt = true
	}
	return info, nil
}
