package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNPMLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The registry escapes the "/" in a scoped package name as %2f, so
		// match on the raw (still-escaped) path, not the decoded one.
		switch r.URL.EscapedPath() {
		case "/lodash":
			w.Write([]byte(`{"time":{"created":"2012-04-23T16:37:11.912Z","modified":"2026-01-01T00:00:00.000Z"}}`))
		case "/@scope%2fpkg":
			w.Write([]byte(`{"time":{"created":"2026-09-29T00:00:00.000Z"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	npm := &NPM{BaseURL: srv.URL, Client: srv.Client()}

	info, err := npm.Lookup(context.Background(), "lodash")
	if err != nil {
		t.Fatalf("Lookup(lodash): %v", err)
	}
	if !info.Exists || !info.HasCreatedAt {
		t.Fatalf("expected lodash to exist with a created date, got %+v", info)
	}
	want, _ := time.Parse(time.RFC3339, "2012-04-23T16:37:11.912Z")
	if !info.CreatedAt.Equal(want) {
		t.Errorf("CreatedAt = %v, want %v", info.CreatedAt, want)
	}

	scoped, err := npm.Lookup(context.Background(), "@scope/pkg")
	if err != nil {
		t.Fatalf("Lookup(@scope/pkg): %v", err)
	}
	if !scoped.Exists {
		t.Error("expected scoped package to exist")
	}

	missing, err := npm.Lookup(context.Background(), "does-not-exist")
	if err != nil {
		t.Fatalf("Lookup(does-not-exist): %v", err)
	}
	if missing.Exists {
		t.Error("expected does-not-exist to not exist")
	}
}

func TestNPMUnreachable(t *testing.T) {
	npm := &NPM{BaseURL: "http://127.0.0.1:1", Client: &http.Client{Timeout: time.Second}}
	_, err := npm.Lookup(context.Background(), "anything")
	if err == nil {
		t.Fatal("expected an error for an unreachable registry")
	}
	if _, ok := err.(*UnreachableError); !ok {
		t.Fatalf("expected *UnreachableError, got %T: %v", err, err)
	}
}

func TestPyPILookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/pypi/requests/json":
			w.Write([]byte(`{"releases":{"0.2.0":[],"0.10.0":[{"upload_time_iso_8601":"2012-01-22T05:08:17.091441Z"}],"2.31.0":[{"upload_time_iso_8601":"2023-05-22T15:12:33.033734Z"}]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	pypi := &PyPI{BaseURL: srv.URL, Client: srv.Client()}
	info, err := pypi.Lookup(context.Background(), "requests")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if !info.Exists || !info.HasCreatedAt {
		t.Fatalf("expected requests to exist with a created date, got %+v", info)
	}
	want, _ := time.Parse(time.RFC3339, "2012-01-22T05:08:17.091441Z")
	if !info.CreatedAt.Equal(want) {
		t.Errorf("CreatedAt = %v, want earliest release %v", info.CreatedAt, want)
	}
}

func TestCratesLookup(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		switch r.URL.Path {
		case "/api/v1/crates/serde":
			w.Write([]byte(`{"crate":{"created_at":"2014-12-05T20:20:39.487502Z"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	crates := &Crates{BaseURL: srv.URL, Client: srv.Client(), UserAgent: "installwall-test (contact: test@example.invalid)"}
	info, err := crates.Lookup(context.Background(), "serde")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if !info.Exists || !info.HasCreatedAt {
		t.Fatalf("expected serde to exist with a created date, got %+v", info)
	}
	if gotUA == "" {
		t.Error("crates.io requires a descriptive User-Agent, got none")
	}
}

func TestRubyGemsLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/gems/rails.json":
			w.Write([]byte(`{"name":"rails","version":"7.1.0"}`))
		case "/api/v1/versions/rails.json":
			w.Write([]byte(`[{"number":"7.1.0","created_at":"2026-01-01T00:00:00.000Z"},{"number":"0.8.0","created_at":"2009-07-25T18:01:56.000Z"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	rg := &RubyGems{BaseURL: srv.URL, Client: srv.Client()}
	info, err := rg.Lookup(context.Background(), "rails")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if !info.Exists || !info.HasCreatedAt {
		t.Fatalf("expected rails to exist with a created date, got %+v", info)
	}
	want, _ := time.Parse(time.RFC3339, "2009-07-25T18:01:56.000Z")
	if !info.CreatedAt.Equal(want) {
		t.Errorf("CreatedAt = %v, want earliest version %v", info.CreatedAt, want)
	}
}
