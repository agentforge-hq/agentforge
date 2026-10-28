package agentforge

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchCLIForwardsOnlyExplicitConstraints(t *testing.T) {
	for _, args := range [][]string{
		{"--available-tool", "Go", "--available-tool", "git", "--available-tool", "go", "--allow-setup=false", "--allow-credentials=false"},
		{"--no-tools"},
		{"--allow-setup=true"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if len(body) != 4 || body["query"] != "prepare a report" || body["harness"] != "codex" {
					t.Error("unexpected uploaded fields", body)
				}
				constraints := body["constraints"].(map[string]any)
				switch args[0] {
				case "--available-tool":
					tools := constraints["available_tools"].([]any)
					if len(tools) != 2 || tools[0] != "git" || tools[1] != "go" || constraints["allow_setup"] != false || constraints["allow_credentials"] != false {
						t.Error(constraints)
					}
				case "--no-tools":
					if len(constraints["available_tools"].([]any)) != 0 || constraints["allow_setup"] != true || constraints["allow_credentials"] != true {
						t.Error(constraints)
					}
				default:
					if constraints["available_tools"] != nil || constraints["allow_setup"] != true || constraints["allow_credentials"] != true {
						t.Error(constraints)
					}
				}
				w.Write([]byte(`{"rows":[],"meta":{"status":"no_match","retrieval_mode":"hybrid"}}`))
			}))
			defer server.Close()
			var stdout, stderr bytes.Buffer
			arguments := []string{"search", "--query", "prepare a report", "--harness", "codex", "--endpoint", server.URL}
			if code := Run(append(arguments, args...), &stdout, &stderr); code != 0 {
				t.Fatal(code, stderr.String())
			}
		})
	}
}

func TestSearchRejectsPrivateOrOversizedToolInputLocally(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	for _, names := range [][]string{{"/Users/private/bin/tool"}, {"token=private"}, {strings.Repeat("a", 65)}, strings.Split(strings.Repeat("tool,", 32)+"tool", ",")} {
		_, err := CloudSearchWithConstraints("task", "codex", server.URL, 3, &SearchConstraints{AvailableTools: names, AllowSetup: true, AllowCredentials: true})
		reject(t, err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"search", "--query", "task", "--endpoint", server.URL, "--no-tools", "--available-tool", "git"}, &stdout, &stderr); code != 2 {
		t.Fatal(code)
	}
	if calls != 0 {
		t.Fatal("invalid constraints reached the network")
	}
}

func TestCloudSearchPreservesRichResultEvidence(t *testing.T) {
	body := `{"rows":[{"name":"example","profile":{"extractor_version":"v2","tools":["git"],"evidence":[{"quote":"Inspect reports","status":"declared"}]},"fit":{"reason":"Inspect reports","notes":["Support unknown"]},"harness_support":[{"harness":"codex","status":"unknown"}],"source":{"files":[{"path":"SKILL.md","git_blob_sha1":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}],"discovered_via":["registry"]}}],"meta":{"status":"matches","retrieval_mode":"hybrid"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
	defer server.Close()
	result, err := CloudSearch("task", "codex", server.URL, 3)
	must(t, err)
	row := result["rows"].([]any)[0].(M)
	if row["profile"].(M)["extractor_version"] != "v2" || row["fit"].(M)["reason"] != "Inspect reports" || row["harness_support"].([]any)[0].(M)["status"] != "unknown" || row["source"].(M)["discovered_via"].([]any)[0] != "registry" {
		t.Fatal(result)
	}
}

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
