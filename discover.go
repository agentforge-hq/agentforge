package agentforge

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var allowedHosts = map[string]bool{"api.github.com": true, "github.com": true, "codeload.github.com": true, "raw.githubusercontent.com": true, "skills.sh": true, "www.skills.sh": true}

type injectionPattern struct {
	Code, Severity string
	Pattern        *regexp.Regexp
}

var injectionPatterns = []injectionPattern{
	{"instruction-override", "high", regexp.MustCompile(`(?i)ignore\s+(?:(?:all|any)\s+)?(?:previous|prior|system|developer)\s+(?:instructions|rules|messages)`)},
	{"role-spoofing", "high", regexp.MustCompile(`(?i)<\|(?:im_start|system|developer)\|>|\[INST\]|<system>`)},
	{"secret-request", "high", regexp.MustCompile(`(?i)(?:reveal|print|upload|send|exfiltrat\w*)[^\n]{0,80}(?:api.key|secret|password|credential|private.key)`)},
	{"disable-controls", "high", regexp.MustCompile(`(?i)(?:disable|bypass|ignore)[^\n]{0,50}(?:safety|approval|sandbox|security.check)`)},
	{"remote-execution", "high", regexp.MustCompile(`(?i)(?:curl|wget)[^\n]*\|\s*(?:sh|bash|python)`)},
	{"upload-command", "medium", regexp.MustCompile(`(?i)curl[^\n]*(?:--data|-d\s|--upload-file|-T\s)|requests\.post\(`)},
	{"obfuscated-execution", "high", regexp.MustCompile(`(?i)(?:exec|eval)\s*\([^\n]*(?:b64decode|fromhex)|base64\s+-d[^\n]*\|`)},
	{"bidi-control", "medium", regexp.MustCompile("[\u202a-\u202e\u2066-\u2069]")},
}

func (e *Engine) Inspect(root string) (M, error) {
	root, err := realPath(root)
	if err != nil {
		return nil, err
	}
	fingerprint, err := e.treeDigest(root)
	if err != nil {
		return nil, err
	}
	files, err := e.portableFiles(root, false)
	if err != nil {
		return nil, err
	}
	findings, unscanned := []M{}, []M{}
	for _, file := range files {
		relative, _ := filepath.Rel(root, file)
		relative = filepath.ToSlash(relative)
		text, err := readText(file, e.Config.Limits["file_bytes"])
		if err != nil || strings.ContainsRune(text, 0) {
			reason := "binary data"
			if err != nil {
				reason = err.Error()
			}
			unscanned = append(unscanned, M{"file": relative, "reason": reason})
			continue
		}
		for number, line := range strings.Split(text, "\n") {
			for _, pattern := range injectionPatterns {
				if pattern.Pattern.MatchString(line) {
					evidence := []rune(strings.TrimSpace(line))
					if len(evidence) > 180 {
						evidence = evidence[:180]
					}
					findings = append(findings, M{"code": pattern.Code, "severity": pattern.Severity, "file": relative, "line": number + 1, "evidence": string(evidence)})
				}
			}
		}
	}
	return M{"findings": findings, "unscanned": unscanned, "tree_sha256": fingerprint, "executed": false, "assessment": "Heuristic triage only; no findings does not establish trust or rule out injection"}, nil
}
func checkedURL(raw string) error {
	value, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if value.Scheme != "https" || !allowedHosts[strings.ToLower(value.Hostname())] || (value.Port() != "" && value.Port() != "443") || value.User != nil {
		return errors.New("only HTTPS public GitHub and skills.sh hosts are allowed")
	}
	return nil
}
func (e *Engine) download(raw string, limit int) ([]byte, error) {
	if err := checkedURL(raw); err != nil {
		return nil, err
	}
	client := http.Client{Timeout: operationTimeout}
	if e.Client != nil {
		client = *e.Client
		client.Timeout = operationTimeout
	}
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		return checkedURL(request.URL.String())
	}
	request, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "agentforge/"+Version)
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("public fetch failed: %w", err)
	}
	defer response.Body.Close()
	if err = checkedURL(response.Request.URL.String()); err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("public fetch failed: HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("download exceeds %d bytes", limit)
	}
	return data, nil
}

type SourceSelection struct {
	Owner, Repo, Ref string
	SkillPath        *string
	Slug             string
}

func safeRelative(name string) (string, error) {
	if !utf8.ValidString(name) {
		return "", errors.New("archive paths must be valid UTF-8")
	}
	if name == "" || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\:\x00") {
		return "", fmt.Errorf("unsafe archive/path entry: %s", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == ".." || part == "." {
			return "", fmt.Errorf("unsafe archive/path entry: %s", name)
		}
	}
	if path.Clean(name) != name {
		return "", fmt.Errorf("noncanonical archive/path entry: %s", name)
	}
	return name, nil
}
func parseSource(source, ref string, skillPath *string) (SourceSelection, error) {
	result := SourceSelection{Ref: ref, SkillPath: skillPath}
	if strings.Contains(source, "://") {
		if err := checkedURL(source); err != nil {
			return result, err
		}
		value, err := url.Parse(source)
		if err != nil {
			return result, err
		}
		pieces := strings.Split(strings.Trim(value.EscapedPath(), "/"), "/")
		for i, piece := range pieces {
			pieces[i], err = url.PathUnescape(piece)
			if err != nil {
				return result, err
			}
		}
		switch strings.ToLower(value.Hostname()) {
		case "skills.sh", "www.skills.sh":
			if len(pieces) != 3 {
				return result, errors.New("use a skills.sh/owner/repo/skill-slug URL")
			}
			result.Owner, result.Repo, result.Slug = pieces[0], pieces[1], pieces[2]
		case "github.com":
			if len(pieces) < 2 {
				return result, errors.New("GitHub URL must name an owner and repository")
			}
			result.Owner, result.Repo = pieces[0], pieces[1]
			if len(pieces) > 2 {
				if len(pieces) < 4 || !contains([]string{"tree", "blob"}, pieces[2]) {
					return result, errors.New("use a repository URL or /tree/REF/path URL")
				}
				if result.Ref != "" && result.Ref != pieces[3] {
					return result, errors.New("for refs containing slashes use a repository URL with --ref and --skill-path")
				}
				result.Ref = pieces[3]
				if result.SkillPath == nil && len(pieces) > 4 {
					selected := strings.TrimSuffix(strings.Join(pieces[4:], "/"), "/SKILL.md")
					result.SkillPath = &selected
				}
			}
		default:
			return result, errors.New("source must be a GitHub repository or skills.sh skill page")
		}
	} else {
		pieces := strings.Split(strings.Trim(source, "/"), "/")
		if len(pieces) < 2 {
			return result, errors.New("expected owner/repo[/path]")
		}
		result.Owner, result.Repo = pieces[0], pieces[1]
		if result.SkillPath == nil && len(pieces) > 2 {
			selected := strings.Join(pieces[2:], "/")
			result.SkillPath = &selected
		}
	}
	result.Repo = strings.TrimSuffix(result.Repo, ".git")
	name := regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	if !name.MatchString(result.Owner) || !name.MatchString(result.Repo) || contains([]string{".", ".."}, result.Owner) || contains([]string{".", ".."}, result.Repo) {
		return result, errors.New("invalid GitHub repository name")
	}
	if result.SkillPath != nil {
		if _, err := safeRelative(*result.SkillPath); err != nil {
			return result, err
		}
	}
	if result.Ref == "" {
		result.Ref = "HEAD"
	}
	return result, nil
}
func readZipMember(file *zip.File, limit int) ([]byte, error) {
	if file.UncompressedSize64 > uint64(limit) {
		return nil, errors.New("ZIP member exceeds read limit")
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if uint64(len(data)) != file.UncompressedSize64 || len(data) > limit {
		return nil, errors.New("ZIP member size mismatch")
	}
	return data, nil
}
func (e *Engine) unpackSkill(data []byte, selection SourceSelection, destination string) (string, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", errors.New("invalid GitHub ZIP archive")
	}
	if len(archive.File) > e.Config.Limits["tree_files"] {
		return "", errors.New("repository archive has too many entries")
	}
	roots, names := map[string]bool{}, map[string]bool{}
	for _, file := range archive.File {
		name := strings.TrimSuffix(file.Name, "/")
		if _, err = safeRelative(name); err != nil {
			return "", err
		}
		if names[name] {
			return "", errors.New("duplicate ZIP entry")
		}
		names[name] = true
		roots[strings.Split(name, "/")[0]] = true
	}
	if len(roots) != 1 {
		return "", errors.New("expected a single GitHub archive root")
	}
	prefix := sortedKeys(roots)[0] + "/"
	candidates := []string{}
	for _, file := range archive.File {
		if !strings.HasSuffix(file.Name, "/SKILL.md") {
			continue
		}
		relative := strings.TrimSuffix(strings.TrimSuffix(strings.TrimPrefix(file.Name, prefix), "SKILL.md"), "/")
		if selection.SkillPath != nil && relative != strings.TrimSuffix(*selection.SkillPath, "/") {
			continue
		}
		if selection.Slug != "" {
			content, err := readZipMember(file, e.Config.Limits["file_bytes"])
			if err != nil {
				return "", err
			}
			fm := parseFrontmatter(strings.TrimPrefix(string(content), "\ufeff"))
			if len(fm.Problems) > 0 || str(fm.Fields["name"]) != selection.Slug {
				continue
			}
		}
		candidates = append(candidates, relative)
	}
	if len(candidates) != 1 {
		return "", fmt.Errorf("expected exactly one skill, found %d; specify --skill-path", len(candidates))
	}
	selected := prefix
	if candidates[0] != "" {
		selected += candidates[0] + "/"
	}
	total := uint64(0)
	for _, file := range archive.File {
		inside := strings.HasPrefix(file.Name, selected)
		ancestor := strings.HasPrefix(selected, strings.TrimSuffix(file.Name, "/")+"/")
		if !inside && !ancestor {
			continue
		}
		mode := file.Mode()
		if mode&os.ModeSymlink != 0 || (!mode.IsRegular() && !mode.IsDir()) {
			return "", errors.New("selected skill contains a symlink or special file")
		}
		if inside {
			if file.UncompressedSize64 > uint64(e.Config.Limits["file_bytes"]) {
				return "", errors.New("selected skill member exceeds expanded size limit")
			}
			total += file.UncompressedSize64
			if total > uint64(e.Config.Limits["extract_bytes"]) {
				return "", errors.New("selected skill exceeds expanded size limits")
			}
		}
	}
	if err = os.Mkdir(destination, 0700); err != nil {
		return "", err
	}
	for _, file := range archive.File {
		if file.FileInfo().IsDir() || !strings.HasPrefix(file.Name, selected) {
			continue
		}
		relative, err := safeRelative(strings.TrimPrefix(file.Name, selected))
		if err != nil {
			return "", err
		}
		target := filepath.Join(destination, filepath.FromSlash(relative))
		if !within(target, destination) {
			return "", errors.New("ZIP entry escapes destination")
		}
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return "", err
		}
		payload, err := readZipMember(file, e.Config.Limits["file_bytes"])
		if err != nil {
			return "", err
		}
		mode := os.FileMode(0644)
		if file.Mode().Perm()&0111 != 0 {
			mode = 0755
		}
		if err = writeNew(target, payload, mode); err != nil {
			return "", err
		}
		if err = os.Chmod(target, mode); err != nil {
			return "", err
		}
	}
	return candidates[0], nil
}
func (e *Engine) Fetch(source, state, ref string, skillPath *string, expectedArchive string) (M, error) {
	selection, err := parseSource(source, ref, skillPath)
	if err != nil {
		return nil, err
	}
	commitURL := "https://api.github.com/repos/" + selection.Owner + "/" + selection.Repo + "/commits/" + url.PathEscape(selection.Ref)
	data, err := e.download(commitURL, e.Config.Limits["file_bytes"])
	if err != nil {
		return nil, err
	}
	commit := M{}
	if err = json.Unmarshal(data, &commit); err != nil {
		return nil, err
	}
	sha := str(commit["sha"])
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(sha) {
		return nil, errors.New("GitHub did not return a pinned commit SHA")
	}
	data, err = e.download("https://codeload.github.com/"+selection.Owner+"/"+selection.Repo+"/zip/"+sha, e.Config.Limits["download_bytes"])
	if err != nil {
		return nil, err
	}
	archiveHash := hashBytes(data)
	if expectedArchive != "" && archiveHash != expectedArchive {
		return nil, errors.New("archive checksum does not match --archive-sha256")
	}
	quarantine := filepath.Join(state, "quarantine")
	if err = os.MkdirAll(quarantine, 0700); err != nil {
		return nil, err
	}
	staging, err := os.MkdirTemp(quarantine, ".fetch-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(staging)
	selected, err := e.unpackSkill(data, selection, filepath.Join(staging, "skill"))
	if err != nil {
		return nil, err
	}
	text, err := readText(filepath.Join(staging, "skill", "SKILL.md"), e.Config.Limits["file_bytes"])
	if err != nil {
		return nil, err
	}
	fm := parseFrontmatter(text)
	name := str(fm.Fields["name"])
	if len(fm.Problems) > 0 || !validName(name) {
		return nil, errors.New("downloaded skill has invalid or unsupported metadata")
	}
	inspection, err := e.Inspect(filepath.Join(staging, "skill"))
	if err != nil {
		return nil, err
	}
	manifest := M{"source": source, "repository": selection.Owner + "/" + selection.Repo, "commit": sha, "skill_path": selected, "name": name, "archive_sha256": archiveHash, "tree_sha256": inspection["tree_sha256"], "inspection": inspection, "status": "quarantined", "executed": false}
	if err = atomicJSON(filepath.Join(staging, "manifest.json"), manifest); err != nil {
		return nil, err
	}
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	destination := filepath.Join(quarantine, name+"-"+id)
	if err = publishDirectory(staging, destination); err != nil {
		return nil, err
	}
	manifest["quarantine"] = destination
	return manifest, nil
}
func (e *Engine) Search(query string) (M, error) {
	parameters := url.Values{"q": {query + " agent skills"}, "per_page": {"10"}}
	data, err := e.download("https://api.github.com/search/repositories?"+parameters.Encode(), e.Config.Limits["file_bytes"])
	if err != nil {
		return nil, err
	}
	var response struct {
		Items []struct {
			FullName    string  `json:"full_name"`
			URL         string  `json:"html_url"`
			Description *string `json:"description"`
		} `json:"items"`
	}
	if err = json.Unmarshal(data, &response); err != nil {
		return nil, err
	}
	repositories := []M{}
	for i, item := range response.Items {
		if i >= 10 {
			break
		}
		repositories = append(repositories, M{"name": item.FullName, "url": item.URL, "description": item.Description})
	}
	return M{"repositories": repositories, "skills_sh_catalog": "https://skills.sh/"}, nil
}
