package agentforge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCloudSearchContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("invalid request method or content type")
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 3 || body["query"] != "write a launch plan" || body["harness"] != "codex" {
			t.Error("unexpected uploaded fields", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"rows":[],"meta":{"status":"no_match","retrieval_mode":"hybrid"}}`))
	}))
	defer server.Close()
	if _, err := CloudSearch("write a launch plan", "codex", server.URL, 3); err != nil {
		t.Fatal(err)
	}
}
func TestCloudSearchRejectsInvalidRequest(t *testing.T) {
	for _, query := range []string{"", strings.Repeat("é", 1025)} {
		if _, err := CloudSearch(query, "codex", "https://example.com", 3); err == nil {
			t.Error("accepted invalid query")
		}
	}
	for _, endpoint := range []string{"http://example.com", "https://user:password@example.com", "file:///tmp/catalog"} {
		if _, err := CloudSearch("task", "codex", endpoint, 3); err == nil {
			t.Error("accepted invalid endpoint")
		}
	}
	if _, err := CloudSearch("task", "unsupported", "https://example.com", 3); err == nil {
		t.Error("accepted harness")
	}
	if _, err := CloudSearch("task", "codex", "https://example.com", 21); err == nil {
		t.Error("accepted limit")
	}
}
func TestCloudSearchRejectsBadResponses(t *testing.T) {
	for _, body := range []string{`not json`, `{"rows":{},"meta":{}}`, `{"rows":[],"meta":{"status":"ok"}}`, strings.Repeat("x", 1048577)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		if _, err := CloudSearch("task", "", server.URL, 3); err == nil {
			t.Error("accepted bad response")
		}
		server.Close()
	}
}
func TestCloudSearchDoesNotExposeServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		w.Write([]byte("private diagnostic"))
	}))
	defer server.Close()
	_, err := CloudSearch("task", "", server.URL, 3)
	if err == nil || strings.Contains(err.Error(), "private diagnostic") {
		t.Fatal("unsafe error", err)
	}
}
func TestCloudSearchDoesNotFollowRedirects(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer source.Close()
	if _, err := CloudSearch("task", "", source.URL, 3); err == nil || called {
		t.Fatal("followed redirect")
	}
}
