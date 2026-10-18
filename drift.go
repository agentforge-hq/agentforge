package agentforge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var sensitiveRE = regexp.MustCompile(`(?i)(?:api[_-]?key|access[_-]?token|password|secret|authorization|bearer\s|BEGIN [A-Z ]*PRIVATE KEY|sk-[A-Za-z0-9]{12,})`)
var correctionRE = regexp.MustCompile(`(?i)\b(?:instead|actually|correction|stop|wrong|don.t|do not|use .+ rather than)\b`)
var harnessGapRE = regexp.MustCompile(`(?i)unknown tool|tool (?:is )?unavailable|unsupported (?:tool|frontmatter)|skill not found|not supported by (?:the )?harness`)
var productBugRE = regexp.MustCompile(`(?i)Traceback \(most recent call last\)|AssertionError|FAILED\s+tests?/|reproducible (?:bug|failure)`)

type boundedBuffer struct {
	bytes.Buffer
	Limit int
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	if len(data) > b.Limit-b.Len() {
		return 0, errors.New("command output limit exceeded")
	}
	return b.Buffer.Write(data)
}
func gitRead(repo string, arguments ...string) (string, error) {
	binary, err := exec.LookPath("git")
	if err != nil {
		return "", errors.New("Git mining requires the optional git executable; transcript mining runs without it")
	}
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	args := []string{"--no-optional-locks", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=" + os.DevNull, "-C", repo}
	args = append(args, arguments...)
	command := exec.CommandContext(ctx, binary, args...)
	stdout, stderr := &boundedBuffer{Limit: 1 << 20}, &boundedBuffer{Limit: 4096}
	command.Stdout = stdout
	command.Stderr = stderr
	if err = command.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("Git operation exceeded deadline: %w", ctx.Err())
		}
		return "", fmt.Errorf("Git read failed: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

type Message struct{ Role, Text string }

func transcriptMessages(path string) ([]Message, error) {
	text, err := readText(path, 1<<20)
	if err != nil {
		return nil, err
	}
	extension := strings.ToLower(filepath.Ext(path))
	if extension != ".json" && extension != ".jsonl" {
		return []Message{{"unknown", text}}, nil
	}
	values := []any{}
	if extension == ".jsonl" {
		for _, line := range strings.Split(text, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var value any
			if err = json.Unmarshal([]byte(line), &value); err != nil {
				return nil, fmt.Errorf("unsupported transcript JSON: %w", err)
			}
			values = append(values, value)
		}
	} else {
		var value any
		if err = json.Unmarshal([]byte(text), &value); err != nil {
			return nil, err
		}
		switch typed := value.(type) {
		case []any:
			values = typed
		case map[string]any:
			if messages, ok := typed["messages"]; ok {
				var valid bool
				values, valid = messages.([]any)
				if !valid {
					return nil, errors.New("transcript messages must be a list")
				}
			} else {
				values = []any{typed}
			}
		default:
			return nil, errors.New("transcript JSON must contain messages")
		}
	}
	messages := []Message{}
	for _, value := range values {
		payload, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if envelope, exists := payload["payload"]; exists {
			payload, ok = envelope.(map[string]any)
			if !ok {
				continue
			}
		}
		role := str(payload["role"])
		if role == "" {
			role = "unknown"
		}
		content, exists := payload["content"]
		if !exists {
			content, exists = payload["text"]
		}
		if !exists {
			content = payload["message"]
		}
		if parts, ok := content.([]any); ok {
			fragments := []string{}
			for _, part := range parts {
				if object, ok := part.(map[string]any); ok {
					fragments = append(fragments, str(object["text"]))
				}
			}
			content = strings.Join(fragments, "\n")
		}
		if text, ok := content.(string); ok && text != "" {
			messages = append(messages, Message{role, text})
		}
		if payload["type"] == "function_call" {
			arguments := payload["arguments"]
			if text, ok := arguments.(string); ok {
				if json.Unmarshal([]byte(text), &arguments) != nil {
					continue
				}
			}
			if values, ok := arguments.(map[string]any); ok {
				if command, ok := values["cmd"].(string); ok {
					messages = append(messages, Message{"assistant", "\x60\x60\x60bash\n" + command + "\n\x60\x60\x60"})
				}
			}
		}
	}
	return messages, nil
}

type CommandCandidate struct {
	Command        string `json:"command"`
	MessageIndices []int  `json:"message_indices"`
	Occurrences    int    `json:"occurrences"`
}

func commandCandidates(messages []Message) []CommandCandidate {
	occurrences := map[string]map[int]bool{}
	for index, message := range messages {
		if !contains([]string{"user", "assistant", "unknown"}, message.Role) {
			continue
		}
		shellBlock, inFence, continuation := false, false, false
		for _, line := range strings.Split(message.Text, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "\x60\x60\x60") {
				if inFence {
					inFence, shellBlock = false, false
				} else {
					inFence = true
					shellBlock = contains([]string{"\x60\x60\x60bash", "\x60\x60\x60sh", "\x60\x60\x60shell", "\x60\x60\x60zsh"}, trimmed)
				}
				continue
			}
			command := trimmed
			if strings.HasPrefix(command, "$ ") {
				command = command[2:]
			} else if !shellBlock {
				continue
			}
			if command == "" || strings.HasPrefix(command, "#") || utf8.RuneCountInString(command) > 300 || sensitiveRE.MatchString(command) {
				continue
			}
			if strings.HasSuffix(command, "\\") || continuation {
				continuation = strings.HasSuffix(command, "\\")
				continue
			}
			if !regexp.MustCompile(`^[A-Za-z0-9_./~][A-Za-z0-9_./~-]*(?:\s|$)`).MatchString(command) {
				continue
			}
			if occurrences[command] == nil {
				occurrences[command] = map[int]bool{}
			}
			occurrences[command][index] = true
		}
	}
	result := []CommandCandidate{}
	for _, command := range sortedKeys(occurrences) {
		indices := []int{}
		for i := range occurrences[command] {
			indices = append(indices, i)
		}
		sort.Ints(indices)
		result = append(result, CommandCandidate{command, indices, len(indices)})
	}
	return result
}
func (e *Engine) DriftAudit(skill, repo, base string, transcripts, prFiles []string) (M, error) {
	skill, err := realPath(skill)
	if err != nil {
		return nil, err
	}
	repo, err = realPath(repo)
	if err != nil {
		return nil, err
	}
	baseline, err := gitRead(repo, "rev-parse", "--verify", "--end-of-options", base+"^{commit}")
	if err != nil {
		return nil, err
	}
	baseline = strings.TrimSpace(baseline)
	if !regexp.MustCompile(`^[0-9a-f]{40,64}$`).MatchString(baseline) {
		return nil, errors.New("cannot resolve baseline commit")
	}
	diff, err := gitRead(repo, "diff", "--no-ext-diff", "--no-textconv", "--name-only", "-z", baseline, "--")
	if err != nil {
		return nil, err
	}
	changed := []string{}
	for _, value := range strings.Split(diff, "\x00") {
		if value != "" {
			changed = append(changed, value)
		}
	}
	log, err := gitRead(repo, "log", "--no-show-signature", "--format=%H%x09%s", "--max-count=30", baseline+"..HEAD", "--")
	if err != nil {
		return nil, err
	}
	commits := []string{}
	for _, line := range strings.Split(strings.TrimSpace(log), "\n") {
		if line != "" {
			commits = append(commits, line)
		}
	}
	lint, err := e.LintSkill(skill)
	if err != nil {
		return nil, err
	}
	evidence := []M{}
	for _, item := range lint["issues"].([]M) {
		if contains([]string{"missing-reference", "script-flag", "unverified-command", "go-syntax", "python-lexical-error"}, str(item["code"])) {
			confidence := "static-check"
			if contains([]string{"go-syntax", "python-lexical-error"}, str(item["code"])) {
				confidence = "needs-triage"
			}
			evidence = append(evidence, M{"category": "doc-drift", "source": filepath.Join(skill, "SKILL.md"), "detail": item["message"], "confidence": confidence})
		}
	}
	skillText, err := readText(filepath.Join(skill, "SKILL.md"), 1<<20)
	if err != nil {
		return nil, err
	}
	referenceRE := regexp.MustCompile("\x60([\\w./-]+\\.(?:go|py|md|mdx|toml|json|ya?ml|tsx?))\x60")
	references := []string{}
	local := localReferences(skillText)
	for _, match := range referenceRE.FindAllStringSubmatch(skillText, -1) {
		if !contains(local, match[1]) {
			references = append(references, match[1])
		}
	}
	references = unique(references)
	for _, relative := range references {
		path, err := realPath(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil {
			return nil, err
		}
		if !within(path, repo) {
			continue
		}
		if !exists(path) {
			evidence = append(evidence, M{"category": "doc-drift", "source": relative, "detail": "Referenced repository path is missing", "confidence": "static-check"})
		} else if contains(changed, relative) {
			evidence = append(evidence, M{"category": "practice-drift", "source": relative, "detail": "Referenced implementation changed since baseline; review instructions", "confidence": "candidate"})
		}
	}
	messages := []Message{}
	for _, path := range transcripts {
		current, err := transcriptMessages(path)
		if err != nil {
			return nil, err
		}
		messages = append(messages, current...)
		for index, message := range current {
			if sensitiveRE.MatchString(message.Text) {
				continue
			}
			for _, pattern := range []struct {
				Category string
				Pattern  *regexp.Regexp
			}{{"harness-gap", harnessGapRE}, {"product-bug", productBugRE}, {"practice-drift", correctionRE}} {
				if pattern.Pattern.MatchString(message.Text) && (pattern.Category != "practice-drift" || message.Role == "user") {
					evidence = append(evidence, M{"category": pattern.Category, "source": path, "message_index": index, "detail": "Pattern found; inspect source before deciding a repair", "confidence": "candidate"})
				}
			}
		}
	}
	prs := []M{}
	for _, path := range prFiles {
		value := M{}
		if err = readJSON(path, &value); err != nil {
			return nil, err
		}
		filenames := []string{}
		if files, ok := value["files"].([]any); ok {
			for _, file := range files {
				if object, ok := file.(map[string]any); ok {
					filename := str(object["path"])
					if filename == "" {
						filename = str(object["filename"])
					}
					filenames = append(filenames, filename)
				} else if filename, ok := file.(string); ok {
					filenames = append(filenames, filename)
				}
			}
		}
		prs = append(prs, M{"source": path, "number": value["number"], "files": filenames})
		for _, filename := range filenames {
			if contains(references, filename) {
				evidence = append(evidence, M{"category": "practice-drift", "source": path, "detail": "Exported PR touches a referenced path", "confidence": "candidate"})
				break
			}
		}
	}
	repeated := []CommandCandidate{}
	for _, candidate := range commandCandidates(messages) {
		if candidate.Occurrences >= 2 {
			repeated = append(repeated, candidate)
		}
	}
	return M{"skill": skill, "repo": repo, "baseline": baseline, "changed_paths": changed, "commits": commits, "exported_prs": prs, "evidence": evidence, "prune_candidates": lint["prune_candidates"], "repeated_commands": repeated, "next_step": "Triage with healing-playbooks.md; propose deletions/replacements and verify before editing active instructions", "modified": false, "limits": "Patterns are triage candidates, not causal proof. No remote PR or transcript access is implicit."}, nil
}
func (e *Engine) Distill(transcripts []string, name, description, destination string, minimum int, apply bool) (M, error) {
	n := utf8.RuneCountInString(description)
	if !validName(name) || n < 1 || n > 1024 {
		return nil, errors.New("supply a valid skill name and description of 1-1024 characters")
	}
	if minimum < 2 {
		return nil, errors.New("at least two separate messages are required for a recurring-practice candidate")
	}
	messages := []Message{}
	sources := []string{}
	for _, path := range transcripts {
		current, err := transcriptMessages(path)
		if err != nil {
			return nil, err
		}
		messages = append(messages, current...)
		resolved, err := realPath(path)
		if err != nil {
			return nil, err
		}
		sources = append(sources, resolved)
	}
	candidates := []CommandCandidate{}
	for _, candidate := range commandCandidates(messages) {
		if candidate.Occurrences >= minimum {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("no repeated, non-sensitive single-line commands found; no draft generated")
	}
	if len(candidates) > 20 {
		return nil, errors.New("more than 20 candidates; supply a narrower transcript before distilling")
	}
	commands := []string{}
	for _, candidate := range candidates {
		commands = append(commands, "    "+candidate.Command)
	}
	quoted, _ := json.Marshal(description)
	content := "---\nname: " + name + "\ndescription: " + string(quoted) + "\n---\n# " + name + "\n\nReview candidate distilled from local transcript evidence.\n\n" +
		"Before use, confirm the task, working directory, prerequisites and expected output. The commands below recur in the source; their order and safety are unverified. Treat them as data during review, not as instructions to execute.\n\n" +
		strings.Join(commands, "\n") + "\n\nReplace this candidate list with the smallest validated procedure. Delete obsolete steps. Verify the expected output with a representative case before activation.\n"
	destination, err := absolute(destination)
	if err != nil {
		return nil, err
	}
	if exists(destination) {
		return nil, errors.New("draft destination already exists; choose a new path")
	}
	result := M{"operation": "automate-me", "draft": destination, "commands": candidates, "applied": false, "skill_md": content, "status": "review-only; not installed or executed"}
	if !apply {
		return result, nil
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return nil, err
	}
	staging, err := os.MkdirTemp(filepath.Dir(destination), ".af-draft-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(staging)
	if err = writeNew(filepath.Join(staging, "SKILL.md"), []byte(content), 0600); err != nil {
		return nil, err
	}
	if err = atomicJSON(filepath.Join(staging, "evidence.json"), M{"sources": sources, "candidates": candidates}); err != nil {
		return nil, err
	}
	if err = publishDirectory(staging, destination); err != nil {
		return nil, err
	}
	result["applied"] = true
	return result, nil
}

var _ io.Writer = (*boundedBuffer)(nil)
