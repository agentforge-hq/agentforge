package agentforge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultSearchEndpoint = "https://agentforge-five-sandy.vercel.app/api/v1/search"

// CloudSearch sends only the explicit task query and optional harness identifier.
func CloudSearch(query, harness, endpoint string, limit int) (M, error) {
	query = strings.TrimSpace(query)
	if query == "" || len([]byte(query)) > 2048 {
		return nil, errors.New("query must contain 1 to 2048 UTF-8 bytes")
	}
	if limit < 1 || limit > 20 {
		return nil, errors.New("limit must be between 1 and 20")
	}
	switch harness {
	case "", "codex", "claude-code", "cursor", "gemini", "opencode", "devin", "portable":
	default:
		return nil, errors.New("unsupported harness identifier")
	}
	if endpoint == "" {
		endpoint = DefaultSearchEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return nil, errors.New("invalid search endpoint")
	}
	loopback := parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1"
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && loopback) {
		return nil, errors.New("search endpoint requires HTTPS, except for loopback development")
	}
	payload := M{"query": query, "limit": limit}
	if harness != "" {
		payload["harness"] = harness
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "agentforge/"+Version)
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("search redirects are not followed") }}
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("skill search could not be reached; continue the original task or retry later")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skill search returned HTTP %d; no skill was installed", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1048577))
	if err != nil || len(raw) > 1048576 {
		return nil, errors.New("skill search response exceeds the size limit")
	}
	var result M
	if json.Unmarshal(raw, &result) != nil {
		return nil, errors.New("skill search returned invalid JSON")
	}
	rows, ok := result["rows"].([]any)
	if !ok || len(rows) > limit {
		return nil, errors.New("skill search returned an invalid result list")
	}
	meta, ok := result["meta"].(map[string]any)
	if !ok || (meta["status"] != "matches" && meta["status"] != "no_match") || (meta["retrieval_mode"] != "hybrid" && meta["retrieval_mode"] != "lexical_only") {
		return nil, errors.New("skill search returned invalid metadata")
	}
	return result, nil
}
