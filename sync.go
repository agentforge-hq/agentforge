package agentforge

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

func (e *Engine) outsideRoots(state, project string, explicitRoots ...string) (string, error) {
	state, err := realPath(state)
	if err != nil {
		return "", err
	}
	for _, root := range explicitRoots {
		resolved, err := realPath(root)
		if err != nil {
			return "", err
		}
		if within(state, resolved) {
			return "", fmt.Errorf("state/quarantine must be outside skill scan roots: %s", root)
		}
	}
	scopes, err := projectScopes(project)
	if err != nil {
		return "", err
	}
	for _, profile := range e.Config.Harnesses {
		for _, pattern := range profile.SkillRoots {
			for _, scope := range scopes {
				rendered := e.render(pattern, scope)
				roots := []string{rendered}
				if strings.ContainsAny(rendered, "*?[") {
					roots, err = e.expand(pattern, scope)
					if err != nil {
						return "", err
					}
				}
				for _, root := range roots {
					resolved, err := realPath(root)
					if err != nil {
						return "", err
					}
					if within(state, resolved) {
						return "", fmt.Errorf("state/quarantine must be outside skill scan roots: %s", root)
					}
				}
			}
		}
	}
	return state, nil
}
func lockState(state string) (func(), error) {
	if err := os.MkdirAll(state, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(state, ".mutation.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("another mutation or interrupted run may own %s; inspect before removing a stale lock: %w", path, err)
	}
	if _, err = fmt.Fprint(file, os.Getpid()); err != nil {
		file.Close()
		os.Remove(path)
		return nil, err
	}
	if err = file.Close(); err != nil {
		os.Remove(path)
		return nil, err
	}
	return func() { os.Remove(path) }, nil
}
func (e *Engine) nodeDigest(path string) (string, error) {
	if isLink(path) {
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		return hashBytes([]byte("LINK\x00" + target)), nil
	}
	return e.treeDigest(path)
}
func checkedSkill(path string) (string, error) {
	if !utf8.ValidString(path) {
		return "", errors.New("managed skill paths must be valid UTF-8 for restorable JSON journals")
	}
	path, err := absolute(path)
	if err != nil {
		return "", err
	}
	parent, err := realPath(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	path = filepath.Join(parent, filepath.Base(path))
	if !isDir(path) || !isFile(filepath.Join(path, "SKILL.md")) {
		return "", fmt.Errorf("expected a skill directory: %s", path)
	}
	text, err := readText(filepath.Join(path, "SKILL.md"), 1<<20)
	if err != nil {
		return "", err
	}
	fm := parseFrontmatter(text)
	n := utf8.RuneCountInString(str(fm.Fields["description"]))
	if len(fm.Problems) > 0 || !validName(str(fm.Fields["name"])) || n < 1 || n > 1024 {
		return "", fmt.Errorf("cannot manage invalid skill metadata: %s: %v", path, fm.Problems)
	}
	return path, nil
}

type StoredItem struct {
	Source     string `json:"source"`
	ParentReal string `json:"parent_real"`
	SHA256     string `json:"sha256"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	Stored     string `json:"stored,omitempty"`
}
type Journal struct {
	Operation string        `json:"operation"`
	Applied   bool          `json:"applied"`
	Canonical *string       `json:"canonical"`
	Items     []*StoredItem `json:"items"`
	ID        string        `json:"id,omitempty"`
	Status    string        `json:"status,omitempty"`
}

func (e *Engine) Archive(paths []string, state string, apply bool, canonical string) (M, error) {
	if len(paths) == 0 {
		return nil, errors.New("at least one skill is required")
	}
	prepared := []string{}
	seen := map[string]bool{}
	for _, path := range paths {
		if reason := managedReason(path); reason != "" {
			return nil, fmt.Errorf("cannot archive managed skill (%s): %s; use the harness or plugin manager", reason, path)
		}
		checked, err := checkedSkill(path)
		if err != nil {
			return nil, err
		}
		if reason := managedReason(checked); reason != "" {
			return nil, fmt.Errorf("cannot archive managed skill (%s): %s; use the harness or plugin manager", reason, checked)
		}
		if seen[checked] {
			return nil, errors.New("duplicate input paths")
		}
		seen[checked] = true
		prepared = append(prepared, checked)
	}
	paths = prepared
	state, err := realPath(state)
	if err != nil {
		return nil, err
	}
	canonicalHash, canonicalName, canonicalReal := "", "", ""
	if canonical != "" {
		if reason := managedReason(canonical); reason != "" {
			return nil, fmt.Errorf("choose a user-owned canonical source; %s is managed (%s)", canonical, reason)
		}
		canonical, err = checkedSkill(canonical)
		if err != nil {
			return nil, err
		}
		if reason := managedReason(canonical); reason != "" {
			return nil, fmt.Errorf("choose a user-owned canonical source; %s is managed (%s)", canonical, reason)
		}
		canonicalHash, err = e.treeDigest(canonical)
		if err != nil {
			return nil, err
		}
		content, err := readText(filepath.Join(canonical, "SKILL.md"), 1<<20)
		if err != nil {
			return nil, err
		}
		canonicalName = str(parseFrontmatter(content).Fields["name"])
		canonicalReal, err = realPath(canonical)
		if err != nil {
			return nil, err
		}
	}
	for _, path := range paths {
		resolved, err := realPath(path)
		if err != nil {
			return nil, err
		}
		if e.SkillDir != "" && within(e.SkillDir, resolved) {
			return nil, errors.New("cannot archive the running agentforge installation")
		}
		if within(state, resolved) {
			return nil, errors.New("cold storage must be outside every moved skill")
		}
		if canonical != "" {
			if within(canonical, path) || (!isLink(path) && within(canonicalReal, resolved)) {
				return nil, errors.New("cannot remove the canonical path, its target or an ancestor")
			}
			if resolved == canonicalReal && !isLink(path) {
				return nil, errors.New("canonical alias through a symlinked parent cannot be archived")
			}
			content, err := readText(filepath.Join(path, "SKILL.md"), 1<<20)
			if err != nil {
				return nil, err
			}
			hash, err := e.treeDigest(path)
			if err != nil {
				return nil, err
			}
			if str(parseFrontmatter(content).Fields["name"]) != canonicalName || hash != canonicalHash {
				return nil, fmt.Errorf("divergent skill; choose or merge it manually: %s; use agentforge compare with --canonical and --copy to inspect differences", path)
			}
			if resolved != canonicalReal {
				if _, err = e.portableFiles(canonical, true); err != nil {
					return nil, fmt.Errorf("distinct trees containing symlinks require manual dependency review: %w", err)
				}
				if _, err = e.portableFiles(path, true); err != nil {
					return nil, err
				}
			}
		}
		for _, other := range paths {
			if other == path {
				continue
			}
			otherReal, err := realPath(other)
			if err != nil {
				return nil, err
			}
			if within(path, other) || (!isLink(other) && within(resolved, otherReal)) {
				return nil, errors.New("nested archive inputs are not supported")
			}
		}
	}
	items := []*StoredItem{}
	for _, path := range paths {
		hash, err := e.nodeDigest(path)
		if err != nil {
			return nil, err
		}
		parent, err := realPath(filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		kind := "directory"
		if isLink(path) {
			kind = "symlink"
		}
		items = append(items, &StoredItem{Source: path, ParentReal: parent, SHA256: hash, Kind: kind, Status: "pending"})
	}
	operation := "cold"
	var canonicalValue *string
	if canonical != "" {
		operation = "dedupe"
		canonicalValue = &canonical
	}
	plan := M{"operation": operation, "applied": false, "canonical": canonicalValue, "items": items}
	if !apply {
		return plan, nil
	}
	unlock, err := lockState(state)
	if err != nil {
		return nil, err
	}
	defer unlock()
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	transaction := filepath.Join(state, "transactions", id)
	if err = os.MkdirAll(transaction, 0700); err != nil {
		return nil, err
	}
	journal := Journal{Operation: operation, Canonical: canonicalValue, Items: items, ID: id, Status: "prepared"}
	for i, item := range items {
		item.Stored = fmt.Sprint(i)
	}
	manifest := filepath.Join(transaction, "manifest.json")
	if err = atomicJSON(manifest, &journal); err != nil {
		return nil, err
	}
	moved := []*StoredItem{}
	perform := func() error {
		for _, item := range items {
			parent, err := realPath(filepath.Dir(item.Source))
			if err != nil {
				return err
			}
			hash, err := e.nodeDigest(item.Source)
			if err != nil {
				return err
			}
			if parent != item.ParentReal || hash != item.SHA256 {
				return fmt.Errorf("source changed after preview: %s", item.Source)
			}
			if canonical != "" {
				current, err := e.treeDigest(canonical)
				if err != nil {
					return err
				}
				if current != canonicalHash {
					return errors.New("canonical skill changed during deduplication")
				}
			}
			if err = e.Rename(item.Source, filepath.Join(transaction, item.Stored)); err != nil {
				return fmt.Errorf("atomic move failed; storage must be on the same filesystem: %w", err)
			}
			moved = append(moved, item)
			item.Status = "stored"
			if err = atomicJSON(manifest, &journal); err != nil {
				return err
			}
		}
		return nil
	}
	if err = perform(); err != nil {
		rollbackProblems := []string{}
		for i := len(moved) - 1; i >= 0; i-- {
			item := moved[i]
			if !exists(item.Source) {
				if undoErr := e.Rename(filepath.Join(transaction, item.Stored), item.Source); undoErr != nil {
					rollbackProblems = append(rollbackProblems, undoErr.Error())
				} else {
					item.Status = "rolled-back"
				}
			}
		}
		journal.Status = "interrupted"
		journalErr := atomicJSON(manifest, &journal)
		return nil, fmt.Errorf("%w; journal retained at %s; rollback errors: %v; journal error: %v", err, manifest, rollbackProblems, journalErr)
	}
	journal.Applied = true
	journal.Status = "stored"
	if err = atomicJSON(manifest, &journal); err != nil {
		return nil, err
	}
	data, _ := json.Marshal(journal)
	result := M{}
	json.Unmarshal(data, &result)
	return result, nil
}

func (e *Engine) Restore(id, state string, apply bool) (M, error) {
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(id) {
		return nil, errors.New("invalid transaction ID")
	}
	state, err := realPath(state)
	if err != nil {
		return nil, err
	}
	transaction := filepath.Join(state, "transactions", id)
	realTransaction, err := realPath(transaction)
	if err != nil {
		return nil, err
	}
	if realTransaction != transaction {
		return nil, errors.New("transaction storage must not be symlinked")
	}
	path := filepath.Join(transaction, "manifest.json")
	var journal Journal
	if err = readJSON(path, &journal); err != nil {
		return nil, err
	}
	if len(journal.Items) == 0 {
		return nil, errors.New("transaction has no items")
	}
	validate := func(item *StoredItem) error {
		if item.Status != "stored" && item.Status != "restored" {
			return errors.New("transaction was interrupted; inspect its journal before recovery")
		}
		if !regexp.MustCompile(`^[0-9]+$`).MatchString(item.Stored) {
			return errors.New("invalid journal storage path")
		}
		if !filepath.IsAbs(item.Source) || !filepath.IsAbs(item.ParentReal) {
			return errors.New("invalid journal source")
		}
		parent, err := realPath(filepath.Dir(item.Source))
		if err != nil {
			return err
		}
		if parent != item.ParentReal {
			return fmt.Errorf("restore parent changed: %s", item.Source)
		}
		node := filepath.Join(transaction, item.Stored)
		if item.Status == "restored" {
			node = item.Source
		} else if exists(item.Source) {
			return fmt.Errorf("refusing to overwrite restore destination: %s", item.Source)
		}
		hash, err := e.nodeDigest(node)
		if err != nil {
			return err
		}
		if hash != item.SHA256 {
			return fmt.Errorf("cold-storage/restored contents changed: %s", node)
		}
		return nil
	}
	for _, item := range journal.Items {
		if err = validate(item); err != nil {
			return nil, err
		}
	}
	result := M{"operation": "restore", "transaction": id, "applied": false, "items": journal.Items}
	if !apply {
		return result, nil
	}
	unlock, err := lockState(state)
	if err != nil {
		return nil, err
	}
	defer unlock()
	for _, item := range journal.Items {
		if err = validate(item); err != nil {
			return nil, err
		}
		if item.Status == "restored" {
			continue
		}
		if err = e.Rename(filepath.Join(transaction, item.Stored), item.Source); err != nil {
			return nil, err
		}
		item.Status = "restored"
		if err = atomicJSON(path, &journal); err != nil {
			return nil, err
		}
	}
	journal.Status = "restored"
	if err = atomicJSON(path, &journal); err != nil {
		return nil, err
	}
	result["applied"] = true
	return result, nil
}

func (e *Engine) portableFiles(source string, includeDirs bool) ([]string, error) {
	result := []string{}
	visited := 0
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == source {
			return nil
		}
		if !utf8.ValidString(path) {
			return errors.New("portable paths must be valid UTF-8")
		}
		visited++
		if visited > e.Config.Limits["tree_files"] {
			return errors.New("portable tree entry limit reached")
		}
		if entry.IsDir() && ignored[entry.Name()] {
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("portable exports cannot contain symlinks: %s", path)
		}
		if entry.IsDir() {
			if includeDirs {
				result = append(result, path)
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("special file in portable tree: %s", path)
		}
		if entry.Name() != ".DS_Store" {
			result = append(result, path)
		}
		return nil
	})
	sort.Strings(result)
	return result, err
}
func publishDirectory(staging, destination string) error {
	if err := os.Mkdir(destination, 0700); err != nil {
		return err
	}
	entries, err := os.ReadDir(staging)
	if err != nil {
		os.Remove(destination)
		return err
	}
	for _, entry := range entries {
		if err = os.Rename(filepath.Join(staging, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
			os.RemoveAll(destination)
			return err
		}
	}
	return nil
}

type ExportOptions struct {
	Apply, Isolate, AllowLossy bool
	Format                     string
}

func (e *Engine) ExportSkill(source, harness, destination string, options ExportOptions) (M, error) {
	return e.exportSkill(source, harness, destination, options, "")
}
func (e *Engine) exportSkill(source, harness, destination string, options ExportOptions, pinned string) (M, error) {
	source, err := checkedSkill(source)
	if err != nil {
		return nil, err
	}
	source, err = realPath(source)
	if err != nil {
		return nil, err
	}
	destination, err = absolute(destination)
	if err != nil {
		return nil, err
	}
	resolved, err := realPath(destination)
	if err != nil {
		return nil, err
	}
	if within(resolved, source) {
		return nil, errors.New("export destination must be outside the source skill")
	}
	if exists(destination) {
		return nil, fmt.Errorf("destination already exists: %s", destination)
	}
	text, err := readText(filepath.Join(source, "SKILL.md"), e.Config.Limits["file_bytes"])
	if err != nil {
		return nil, err
	}
	fm := parseFrontmatter(text)
	if len(fm.Problems) > 0 {
		return nil, fmt.Errorf("invalid frontmatter: %v", fm.Problems)
	}
	manual := contains(e.Config.ManualTargets, harness)
	supported := portableFields
	if harness != "portable" && !manual {
		profile, ok := e.Config.Harnesses[harness]
		if !ok {
			return nil, fmt.Errorf("unknown harness: %s", harness)
		}
		if strings.HasPrefix(profile.Status, "unverified") {
			return nil, errors.New("configure and verify this harness before exporting")
		}
		supported = profile.Frontmatter
	}
	if options.Format == "" {
		options.Format = "skill"
	}
	if !contains([]string{"skill", "cursor-rule"}, options.Format) {
		return nil, errors.New("invalid export format")
	}
	if !manual && options.Format == "skill" && filepath.Base(destination) != str(fm.Fields["name"]) {
		return nil, errors.New("export destination directory must match the declared skill name")
	}
	if options.Isolate && harness != "claude-code" {
		return nil, errors.New("only Claude Code has a verified skill-level isolation mapping (context: fork)")
	}
	dropped := []string{}
	content := "---\n"
	for _, key := range fm.Order {
		if !contains(supported, key) {
			dropped = append(dropped, key)
		} else if !(options.Isolate && key == "context") {
			content += fm.Blocks[key]
		}
	}
	if options.Isolate {
		content += "context: fork\n"
	}
	content += "---\n" + fm.Body
	if pinned != "" {
		dropped = []string{}
	}
	if options.Format == "cursor-rule" {
		if harness != "cursor" || options.Isolate {
			return nil, errors.New("cursor-rule format requires harness cursor")
		}
		description, _ := json.Marshal(str(fm.Fields["description"]))
		content = "---\ndescription: " + string(description) + "\nalwaysApply: false\n---\nRead \x60" + filepath.Join(source, "SKILL.md") + "\x60 when this task applies. Run its scripts from \x60" + source + "\x60.\n"
		dropped = []string{}
		for _, key := range fm.Order {
			if !contains([]string{"name", "description"}, key) {
				dropped = append(dropped, key)
			}
		}
	}
	sort.Strings(dropped)
	files, err := e.portableFiles(source, true)
	if err != nil {
		return nil, err
	}
	relativeFiles := []string{}
	for _, path := range files {
		relative, _ := filepath.Rel(source, path)
		relativeFiles = append(relativeFiles, filepath.ToSlash(relative))
	}
	var isolation any
	if options.Isolate {
		isolation = "context: fork"
	}
	result := M{"operation": "export", "harness": harness, "format": options.Format, "destination": destination, "applied": false, "dropped_fields": dropped, "isolation": isolation, "files": relativeFiles, "manual_upload": manual, "note": "Local export only. Tool permissions and loading semantics remain harness-specific."}
	if !options.Apply {
		return result, nil
	}
	if len(dropped) > 0 && !options.AllowLossy {
		return nil, fmt.Errorf("export drops fields %v; review preview and use --allow-lossy if intended", dropped)
	}
	expected, err := e.treeDigest(source)
	if err != nil {
		return nil, err
	}
	if pinned != "" && expected != pinned {
		return nil, errors.New("source changed after fingerprint review")
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return nil, err
	}
	staging, err := os.MkdirTemp(filepath.Dir(destination), ".af-export-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(staging)
	candidate := ""
	if manual {
		candidate = filepath.Join(staging, "bundle.zip")
		file, err := os.Create(candidate)
		if err != nil {
			return nil, err
		}
		bundle := zip.NewWriter(file)
		for _, path := range files {
			relative, _ := filepath.Rel(source, path)
			info, err := os.Stat(path)
			if err != nil {
				bundle.Close()
				file.Close()
				return nil, err
			}
			name := str(fm.Fields["name"]) + "/" + filepath.ToSlash(relative)
			if info.IsDir() {
				name += "/"
			}
			header := &zip.FileHeader{Name: name, Method: zip.Deflate}
			header.SetMode(info.Mode())
			writer, err := bundle.CreateHeader(header)
			if err != nil {
				bundle.Close()
				file.Close()
				return nil, err
			}
			if info.IsDir() {
				continue
			}
			if filepath.ToSlash(relative) == "SKILL.md" {
				_, err = io.WriteString(writer, content)
			} else {
				input, openErr := os.Open(path)
				if openErr != nil {
					bundle.Close()
					file.Close()
					return nil, openErr
				}
				_, err = io.Copy(writer, input)
				input.Close()
			}
			if err != nil {
				bundle.Close()
				file.Close()
				return nil, err
			}
		}
		if err = bundle.Close(); err != nil {
			file.Close()
			return nil, err
		}
		if err = file.Close(); err != nil {
			return nil, err
		}
	} else if options.Format == "cursor-rule" {
		candidate = filepath.Join(staging, "rule.mdc")
		if err = os.WriteFile(candidate, []byte(content), 0600); err != nil {
			return nil, err
		}
	} else {
		for _, path := range files {
			relative, _ := filepath.Rel(source, path)
			target := filepath.Join(staging, relative)
			info, err := os.Stat(path)
			if err != nil {
				return nil, err
			}
			if info.IsDir() {
				if err = os.MkdirAll(target, 0700); err != nil {
					return nil, err
				}
			} else if err = copyFile(path, target, info.Mode()); err != nil {
				return nil, err
			}
		}
		if pinned == "" {
			if err = os.WriteFile(filepath.Join(staging, "SKILL.md"), []byte(content), 0600); err != nil {
				return nil, err
			}
		}
	}
	current, err := e.treeDigest(source)
	if err != nil {
		return nil, err
	}
	if current != expected {
		return nil, errors.New("source changed during export")
	}
	if pinned != "" {
		copied, err := e.treeDigest(staging)
		if err != nil || copied != pinned {
			return nil, errors.New("copied package differs from the reviewed fingerprint")
		}
	}
	if candidate != "" {
		if err = os.Link(candidate, destination); err != nil {
			return nil, err
		}
	} else if err = publishDirectory(staging, destination); err != nil {
		return nil, err
	}
	result["applied"] = true
	return result, nil
}

func (e *Engine) Promote(quarantine, destination, reviewed string, acceptFindings, apply bool) (M, error) {
	quarantine, err := realPath(quarantine)
	if err != nil {
		return nil, err
	}
	manifest := M{}
	if err = readJSON(filepath.Join(quarantine, "manifest.json"), &manifest); err != nil {
		return nil, err
	}
	source, err := checkedSkill(filepath.Join(quarantine, "skill"))
	if err != nil {
		return nil, err
	}
	digest, err := e.treeDigest(source)
	if err != nil {
		return nil, err
	}
	if digest != str(manifest["tree_sha256"]) {
		return nil, errors.New("quarantine contents changed; fetch/inspect again")
	}
	destination, err = absolute(destination)
	if err != nil {
		return nil, err
	}
	if _, err = os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("destination already exists: %s", destination)
	}
	reviewDigest := digest
	if raw, ok := manifest["catalog_result"]; ok {
		data, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		selected := selectedSkill{}
		if json.Unmarshal(data, &selected) != nil {
			return nil, errors.New("invalid selected source in quarantine manifest")
		}
		if err = selected.validate(str(manifest["prepared_harness"])); err != nil {
			return nil, err
		}
		skillPath := selected.Source.SkillPath
		if skillPath == "." {
			skillPath = ""
		}
		if manifest["commit"] != selected.Source.Revision || manifest["repository"] != selected.Source.Repository || manifest["skill_path"] != skillPath || manifest["prepared_destination"] != destination {
			return nil, errors.New("quarantine source or destination no longer agrees with the selected result; prepare again")
		}
		if err = e.verifyPackage(source, selected); err != nil {
			return nil, err
		}
		binding, err := json.Marshal(M{"tree_sha256": digest, "selected": selected, "destination": destination, "harness": manifest["prepared_harness"], "state": manifest["prepared_state"]})
		if err != nil {
			return nil, err
		}
		reviewDigest = hashBytes(binding)
	}
	inspection, err := e.Inspect(source)
	if err != nil {
		return nil, err
	}
	lint, err := e.LintSkill(source)
	if err != nil {
		return nil, err
	}
	result := M{"operation": "promote", "applied": false, "destination": destination, "tree_sha256": digest, "review_sha256": reviewDigest, "inspection": inspection, "lint_issues": lint["issues"]}
	if selected, ok := manifest["catalog_result"]; ok {
		result["selected"] = selected
		result["removal_preview"] = []string{"agentforge", "sync", "cold", "--skill", destination, "--state-dir", str(manifest["prepared_state"])}
		result["recovery"] = "Preview removal with sync cold, then repeat with --apply. Keep its transaction id and use sync restore --transaction ID with the same --state-dir, preview first and then --apply. Archiving cannot unload instructions already in this conversation."
	}
	if !apply {
		return result, nil
	}
	for _, item := range lint["issues"].([]M) {
		if item["severity"] == "error" && !contains([]string{"name-directory", "unsafe-instruction"}, str(item["code"])) {
			return nil, fmt.Errorf("static validation failed before promotion: %v", item)
		}
	}
	if reviewed != reviewDigest {
		return nil, errors.New("promotion requires --review-sha256 matching the promotion preview fingerprint")
	}
	if (len(inspection["findings"].([]M)) > 0 || len(inspection["unscanned"].([]M)) > 0) && !acceptFindings {
		return nil, errors.New("review findings/unscanned files and use --accept-findings if intended")
	}
	text, err := readText(filepath.Join(source, "SKILL.md"), 1<<20)
	if err != nil {
		return nil, err
	}
	if filepath.Base(destination) != str(parseFrontmatter(text).Fields["name"]) {
		return nil, errors.New("promotion destination directory must match the skill name")
	}
	export, err := e.exportSkill(source, "portable", destination, ExportOptions{Apply: true}, digest)
	if err != nil {
		return nil, err
	}
	result["export"] = export
	result["applied"] = true
	return result, nil
}
