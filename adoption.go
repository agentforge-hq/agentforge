package agentforge

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SourceFile struct {
	Path        string `json:"path"`
	GitBlobSHA1 string `json:"git_blob_sha1"`
}

type CatalogSource struct {
	URL            string       `json:"url"`
	Repository     string       `json:"repository"`
	Revision       string       `json:"revision"`
	SkillPath      string       `json:"skill_path"`
	ManifestSHA256 string       `json:"manifest_sha256"`
	CheckedAt      string       `json:"checked_at"`
	Files          []SourceFile `json:"files,omitempty"`
}

type selectedSkill struct {
	SkillID        string           `json:"skill_id"`
	Name           string           `json:"name"`
	AdoptionStatus string           `json:"adoption_status"`
	Source         CatalogSource    `json:"source"`
	HarnessSupport []HarnessSupport `json:"harness_support,omitempty"`
}

type HarnessSupport struct {
	Harness string `json:"harness"`
	Status  string `json:"status"`
}

var commitSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)
var manifestSHA = regexp.MustCompile(`^[0-9a-f]{64}$`)
var repositoryName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,99}/[A-Za-z0-9][A-Za-z0-9_.-]{0,99}$`)

func (s selectedSkill) validate(harness string) error {
	if !contains([]string{"codex", "claude-code", "cursor", "gemini", "opencode", "devin", "portable"}, harness) {
		return errors.New("choose a supported harness identifier")
	}
	if !validName(s.Name) || s.AdoptionStatus != "review_required" {
		return errors.New("select a named skill with adoption_status review_required")
	}
	if !repositoryName.MatchString(s.Source.Repository) || strings.HasSuffix(s.Source.Repository, ".git") || !commitSHA.MatchString(s.Source.Revision) || !manifestSHA.MatchString(s.Source.ManifestSHA256) {
		return errors.New("selected result requires a public GitHub repository, exact commit SHA and SKILL.md SHA-256")
	}
	if s.Source.SkillPath != "." {
		if _, err := safeRelative(s.Source.SkillPath); err != nil {
			return errors.New("selected result has an unsafe skill directory")
		}
	}
	if s.SkillID != s.Source.Repository+"/"+s.Source.SkillPath {
		return errors.New("skill identity does not agree with the source repository and directory")
	}
	u, err := url.Parse(s.Source.URL)
	expected := "/" + s.Source.Repository + "/tree/" + s.Source.Revision + "/" + s.Source.SkillPath
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != expected {
		return errors.New("source URL must agree exactly with the repository, pinned revision and skill directory")
	}
	if len(s.Source.Files) > 200 {
		return errors.New("selected source file manifest exceeds 200 files")
	}
	seen := map[string]bool{}
	for _, file := range s.Source.Files {
		if _, err := safeRelative(file.Path); err != nil || !commitSHA.MatchString(file.GitBlobSHA1) || seen[file.Path] {
			return errors.New("selected source has an invalid or duplicate package file digest")
		}
		seen[file.Path] = true
	}
	if len(s.Source.Files) > 0 && !seen["SKILL.md"] {
		return errors.New("selected source file manifest must include SKILL.md")
	}
	for _, support := range s.HarnessSupport {
		if support.Harness == harness && support.Status == "incompatible" {
			return errors.New("selected skill is known incompatible with the chosen harness")
		}
	}
	return nil
}

func selectSearchResult(data []byte, index int) (selectedSkill, error) {
	var result struct {
		Rows json.RawMessage `json:"rows"`
	}
	var selected selectedSkill
	if len(data) > 1<<20 || json.Unmarshal(data, &result) != nil || index < 1 || index > 20 {
		return selected, errors.New("provide a search JSON object and a result index between 1 and 20")
	}
	if len(result.Rows) > 0 {
		var rows []json.RawMessage
		if err := json.Unmarshal(result.Rows, &rows); err != nil || len(rows) > 20 || index > len(rows) {
			return selected, errors.New("selected search result does not exist")
		}
		data = rows[index-1]
	} else if index != 1 {
		return selected, errors.New("a single result uses index 1")
	}
	if err := json.Unmarshal(data, &selected); err != nil {
		return selected, errors.New("selected search result has invalid field types")
	}
	return selected, nil
}

func readPackageFile(name string, limit int) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil || len(data) > limit {
		return nil, errors.New("package file exceeds the read limit")
	}
	return data, nil
}

func gitBlobSHA1(data []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", len(data))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func (e *Engine) verifyPackage(root string, selected selectedSkill) error {
	if isLink(root) {
		return errors.New("pinned package root cannot be a symlink")
	}
	wanted := map[string]string{}
	for _, file := range selected.Source.Files {
		wanted[file.Path] = file.GitBlobSHA1
	}
	count, total := 0, 0
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || name == root {
			return err
		}
		count++
		if count > e.Config.Limits["tree_files"] {
			return errors.New("package file count exceeds the limit")
		}
		if entry.Type()&os.ModeSymlink != 0 || ignored[entry.Name()] {
			return errors.New("pinned packages cannot contain symlinks or files omitted by portable export")
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("pinned package contains an unreadable or special file")
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		data, err := readPackageFile(name, e.Config.Limits["file_bytes"])
		if err != nil {
			return err
		}
		total += len(data)
		if total > e.Config.Limits["extract_bytes"] {
			return errors.New("pinned package exceeds the expanded size limit")
		}
		if relative == "SKILL.md" && hashBytes(data) != selected.Source.ManifestSHA256 {
			return errors.New("SKILL.md does not match the selected result; search and inspect again")
		}
		if len(selected.Source.Files) > 0 {
			if digest, ok := wanted[relative]; !ok || gitBlobSHA1(data) != digest {
				return fmt.Errorf("package file does not match the pinned manifest: %s", relative)
			}
			delete(wanted, relative)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(wanted) > 0 {
		return errors.New("pinned package is missing files from the selected manifest")
	}
	data, err := readPackageFile(filepath.Join(root, "SKILL.md"), e.Config.Limits["file_bytes"])
	if err != nil || str(parseFrontmatter(strings.TrimPrefix(string(data), "\ufeff")).Fields["name"]) != selected.Name {
		return errors.New("package name does not agree with the selected result")
	}
	return nil
}

func (e *Engine) installedMatches(selected selectedSkill, project string, roots []string) ([]M, []M, error) {
	inventory, err := e.Scan(project, roots)
	if err != nil {
		return nil, nil, err
	}
	matches := []M{}
	for _, skill := range inventory["skills"].([]M) {
		if skill["name"] != selected.Name {
			continue
		}
		data, err := readPackageFile(str(skill["path"]), e.Config.Limits["file_bytes"])
		sameManifest := err == nil && hashBytes(data) == selected.Source.ManifestSHA256
		matches = append(matches, M{"path": filepath.Dir(str(skill["path"])), "same_manifest": sameManifest, "harnesses": skill["harnesses"], "note": "Name or SKILL.md match only; local origin and activation are not established."})
	}
	return matches, inventory["errors"].([]M), nil
}

// PrepareAdoption fetches into quarantine and previews existing promotion without executing the package.
func (e *Engine) PrepareAdoption(data []byte, index int, destination, harness, state, project string, roots []string) (M, error) {
	selected, err := selectSearchResult(data, index)
	if err != nil {
		return nil, err
	}
	if err = selected.validate(harness); err != nil {
		return nil, err
	}
	destination, err = absolute(destination)
	if err != nil {
		return nil, err
	}
	if filepath.Base(destination) != selected.Name || managedReason(destination) != "" {
		return nil, errors.New("choose a new user-owned directory matching the selected skill name")
	}
	state, err = e.outsideRoots(state, project, roots...)
	if err != nil {
		return nil, err
	}
	resolvedDestination, err := realPath(destination)
	if err != nil {
		return nil, err
	}
	if within(resolvedDestination, state) || within(state, resolvedDestination) {
		return nil, errors.New("adoption destination and external quarantine must not overlap")
	}
	matches, problems, err := e.installedMatches(selected, project, roots)
	if err != nil {
		return nil, err
	}
	plan := M{"operation": "prepare", "applied": false, "selected": selected, "destination": destination, "harness": harness, "installed": matches, "inventory_errors": problems, "executed": false,
		"activation": "Installation does not prove activation. Start a new harness session if the skill is not discoverable. Read the reviewed SKILL.md explicitly for the current task only within its tool permissions. Archiving affects future discovery and cannot unload instructions already in a conversation."}
	if _, err = os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		if err != nil {
			return nil, err
		}
		plan["status"] = "destination_exists"
		return plan, nil
	}
	skillPath := selected.Source.SkillPath
	if skillPath == "." {
		skillPath = ""
	}
	fetched, err := e.fetch(selected.Source.Repository, state, selected.Source.Revision, &skillPath, "", &selected)
	if err != nil {
		return nil, err
	}
	quarantine := str(fetched["quarantine"])
	fetched["prepared_destination"] = destination
	fetched["prepared_harness"] = harness
	fetched["prepared_state"] = state
	if err = atomicJSON(filepath.Join(quarantine, "manifest.json"), fetched); err != nil {
		return nil, err
	}
	preview, err := e.Promote(quarantine, destination, "", false, false)
	if err != nil {
		return nil, err
	}
	plan["status"], plan["quarantine"], plan["promotion"] = "review_required", quarantine, preview
	plan["package_integrity"] = "SKILL.md SHA-256 verified; supporting files are pinned by the fetched repository commit."
	if len(selected.Source.Files) > 0 {
		plan["package_integrity"] = "SKILL.md SHA-256 and complete package Git blob manifest verified."
	}
	plan["promote_command"] = []string{"agentforge", "sync", "promote", "--quarantine", quarantine, "--destination", destination, "--review-sha256", str(preview["review_sha256"]), "--apply"}
	plan["removal_preview"] = []string{"agentforge", "sync", "cold", "--skill", destination, "--state-dir", state}
	plan["recovery"] = "Review sync cold, then repeat with --apply to archive the new installation. Restore using sync restore --transaction ID with the same --state-dir, preview first and then --apply. Existing files are never overwritten."
	return plan, nil
}
