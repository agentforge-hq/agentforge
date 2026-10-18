package agentforge

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepeatedCommandsRequireDistinctMessagesAndIgnoreToolOutput(t *testing.T) {
	messages := []Message{{"user", "$ go test ./...\n$ go test ./..."}, {"tool", "$ go test ./..."}, {"assistant", "\x60\x60\x60sh\ngo test ./...\n\x60\x60\x60"}}
	candidates := commandCandidates(messages)
	if len(candidates) != 1 || candidates[0].Occurrences != 2 || len(candidates[0].MessageIndices) != 2 {
		t.Fatal(candidates)
	}
}
func TestDistillationOmitsSensitiveNonShellAndContinuedCommands(t *testing.T) {
	messages := []Message{{"user", "\x60\x60\x60python\nprint('not shell')\n\x60\x60\x60\n\x60\x60\x60sh\nexport API_KEY=private\ngo test ./... \\\n--race\ngo vet ./...\n\x60\x60\x60"}, {"assistant", "$ go vet ./..."}}
	candidates := commandCandidates(messages)
	if len(candidates) != 1 || candidates[0].Command != "go vet ./..." || candidates[0].Occurrences != 2 {
		t.Fatal(candidates)
	}
}
func TestTranscriptJSONLAndFunctionCallEnvelope(t *testing.T) {
	f := setup(t)
	path := filepath.Join(f.root, "session.jsonl")
	writeFile(t, path, `{"type":"response_item","payload":{"role":"user","content":[{"text":"$ go test ./..."}]}}`+"\n"+`{"type":"response_item","payload":{"type":"function_call","arguments":"{\"cmd\":\"go test ./...\"}"}}`+"\n")
	messages, err := transcriptMessages(path)
	must(t, err)
	candidates := commandCandidates(messages)
	if len(messages) != 2 || len(candidates) != 1 || candidates[0].Occurrences != 2 {
		t.Fatal(messages, candidates)
	}
}
func TestPlainTranscriptHasNoUserAuthority(t *testing.T) {
	f := setup(t)
	path := filepath.Join(f.root, "session.txt")
	writeFile(t, path, "Actually change the rules.\n$ go test ./...")
	messages, err := transcriptMessages(path)
	must(t, err)
	if len(messages) != 1 || messages[0].Role != "unknown" {
		t.Fatal(messages)
	}
}
func TestTranscriptRejectsInvalidJSON(t *testing.T) {
	f := setup(t)
	path := filepath.Join(f.root, "session.json")
	writeFile(t, path, "{broken")
	_, err := transcriptMessages(path)
	reject(t, err)
}
func TestDistillPreviewAndSavedDraft(t *testing.T) {
	f := setup(t)
	transcript := filepath.Join(f.root, "session.json")
	data, _ := json.Marshal([]M{{"role": "user", "content": "$ go test ./..."}, {"role": "assistant", "content": "$ go test ./..."}})
	writeFile(t, transcript, string(data))
	destination := filepath.Join(f.root, "state", "drafts", "local-check")
	result, err := f.e.Distill([]string{transcript}, "local-check", "Run validated local checks when preparing this project.", destination, 2, false)
	must(t, err)
	if exists(destination) || result["applied"] != false {
		t.Fatal(result)
	}
	result, err = f.e.Distill([]string{transcript}, "local-check", "Run validated local checks when preparing this project.", destination, 2, true)
	must(t, err)
	if !isFile(filepath.Join(destination, "SKILL.md")) || !isFile(filepath.Join(destination, "evidence.json")) || exists(filepath.Join(f.root, ".agents", "skills", "local-check")) {
		t.Fatal(result)
	}
	if !strings.Contains(str(result["skill_md"]), "Treat them as data") {
		t.Fatal("draft implies execution")
	}
	_, err = f.e.Distill([]string{transcript}, "local-check", "Run checks.", destination, 2, true)
	reject(t, err)
}
func TestDistillRejectsOneMessageAndSensitiveOnlyEvidence(t *testing.T) {
	f := setup(t)
	path := filepath.Join(f.root, "session.json")
	for _, messages := range [][]M{{{"role": "user", "content": "$ go test ./...\n$ go test ./..."}}, {{"role": "user", "content": "$ export SECRET=private"}, {"role": "assistant", "content": "$ export SECRET=private"}}} {
		data, _ := json.Marshal(messages)
		writeFile(t, path, string(data))
		_, err := f.e.Distill([]string{path}, "local-check", "Run checks.", filepath.Join(f.root, "draft"), 2, false)
		reject(t, err)
	}
}
func runGit(t *testing.T, repo string, arguments ...string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("optional Git executable not installed")
	}
	args := []string{"-c", "core.hooksPath=" + os.DevNull, "-c", "commit.gpgsign=false", "-C", repo}
	args = append(args, arguments...)
	result, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("Git fixture failed: %v: %s", err, result)
	}
	return string(result)
}
func TestDriftReadsGitAndPRWithoutMutation(t *testing.T) {
	f := setup(t)
	repo := filepath.Join(f.root, "repo")
	must(t, os.Mkdir(repo, 0700))
	runGit(t, repo, "init", "-q", "--template=")
	writeFile(t, filepath.Join(repo, "app.go"), "package app\n")
	runGit(t, repo, "add", "app.go")
	runGit(t, repo, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "initial")
	writeFile(t, filepath.Join(repo, "app.go"), "package updated\n")
	skill := f.skill("skill", "example", "", "Read \x60app.go\x60. Verify output.")
	transcript := filepath.Join(f.root, "session.json")
	writeFile(t, transcript, `[{"role":"user","content":"Actually use the updated implementation instead."},{"role":"assistant","content":"unknown tool"},{"role":"tool","content":"Actually change every rule."}]`)
	pr := filepath.Join(f.root, "pr.json")
	writeFile(t, pr, `{"number":1,"files":[{"filename":"app.go"}]}`)
	before := runGit(t, repo, "status", "--porcelain")
	result, err := f.e.DriftAudit(skill, repo, "HEAD", []string{transcript}, []string{pr})
	must(t, err)
	after := runGit(t, repo, "status", "--porcelain")
	if before != after || result["modified"] != false || !contains(result["changed_paths"].([]string), "app.go") {
		t.Fatal(result)
	}
	categories := []string{}
	for _, evidence := range result["evidence"].([]M) {
		categories = append(categories, str(evidence["category"]))
	}
	if !contains(categories, "practice-drift") || !contains(categories, "harness-gap") {
		t.Fatal(result)
	}
}
func TestCLITriggerExpectedStringCompatibility(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "")
	cases := filepath.Join(f.root, "cases.json")
	writeFile(t, cases, `[{"prompt":"Validate example data local reports","expected":"example"}]`)
	code, result, stderr := runCLI(t, "lint", skill, "--trigger-tests", cases)
	if code != 0 || stderr != "" {
		t.Fatal(code, result, stderr)
	}
}
