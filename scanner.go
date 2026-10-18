package agentforge

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (e *Engine) treeDigest(root string) (string, error) {
	if !isDir(root) {
		return "", fmt.Errorf("expected a readable directory: %s", root)
	}
	digest := sha256.New()
	total, count := 0, 0
	writeLength := func(n int) { var data [8]byte; binary.BigEndian.PutUint64(data[:], uint64(n)); digest.Write(data[:]) }
	charge := func(n int) error {
		count++
		total += n
		if count > e.Config.Limits["tree_files"] || total > e.Config.Limits["tree_bytes"] {
			return errors.New("tree fingerprint limits exceeded")
		}
		return nil
	}
	var walk func(string) error
	walk = func(directory string) error {
		entries, err := os.ReadDir(directory)
		if err != nil {
			return err
		}
		dirs := []string{}
		files := []string{}
		for _, entry := range entries {
			path := filepath.Join(directory, entry.Name())
			if entry.IsDir() {
				if !ignored[entry.Name()] {
					dirs = append(dirs, entry.Name())
				}
				continue
			}
			if isLink(path) && isDir(path) && ignored[entry.Name()] {
				continue
			}
			files = append(files, entry.Name())
		}
		for _, name := range dirs {
			if err := charge(0); err != nil {
				return err
			}
			relative, _ := filepath.Rel(root, filepath.Join(directory, name))
			data := []byte(filepath.ToSlash(relative))
			digest.Write([]byte("D"))
			writeLength(len(data))
			digest.Write(data)
		}
		sort.Strings(files)
		for _, name := range files {
			path := filepath.Join(directory, name)
			relative, _ := filepath.Rel(root, path)
			rel := []byte(filepath.ToSlash(relative))
			info, err := os.Lstat(path)
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err != nil {
					return err
				}
				payload := []byte("L" + target)
				if err := charge(len(payload)); err != nil {
					return err
				}
				writeLength(len(rel))
				digest.Write(rel)
				writeLength(len(payload))
				digest.Write(payload)
				continue
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("not a regular file: %s", path)
			}
			if info.Size() > int64(e.Config.Limits["hash_file_bytes"]) {
				return fmt.Errorf("file too large to fingerprint: %s", path)
			}
			header := executableHeader(info.Mode())
			if err := charge(len(header) + int(info.Size())); err != nil {
				return err
			}
			writeLength(len(rel))
			digest.Write(rel)
			writeLength(len(header) + int(info.Size()))
			digest.Write(header)
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			actual, err := io.Copy(digest, io.LimitReader(file, info.Size()+1))
			closeErr := file.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
			after, err := os.Stat(path)
			if err != nil {
				return err
			}
			if actual != info.Size() || !os.SameFile(info, after) || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) {
				return fmt.Errorf("file changed while fingerprinting: %s", path)
			}
		}
		for _, name := range dirs {
			if err := walk(filepath.Join(directory, name)); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func walkSkills(root string, limit int) ([]string, []string) {
	found, problems := []string{}, []string{}
	visited := 0
	var walk func(string, map[string]bool)
	walk = func(current string, ancestors map[string]bool) {
		visited++
		if visited > limit {
			if visited == limit+1 {
				problems = append(problems, "Directory limit reached in "+root)
			}
			return
		}
		real, err := realPath(current)
		if err != nil {
			problems = append(problems, err.Error())
			return
		}
		if ancestors[real] {
			return
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			problems = append(problems, err.Error())
			return
		}
		if isFile(filepath.Join(current, "SKILL.md")) {
			found = append(found, filepath.Join(current, "SKILL.md"))
		}
		next := map[string]bool{}
		for k, v := range ancestors {
			next[k] = v
		}
		next[real] = true
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") || ignored[entry.Name()] {
				continue
			}
			path := filepath.Join(current, entry.Name())
			if isDir(path) {
				walk(path, next)
			}
		}
	}
	walk(root, map[string]bool{})
	sort.Strings(found)
	return found, problems
}

func (e *Engine) skillRecord(path string, cache map[string]M) (M, error) {
	text, err := readText(path, e.Config.Limits["file_bytes"])
	if err != nil {
		return nil, err
	}
	fm := parseFrontmatter(text)
	name := str(fm.Fields["name"])
	if name == "" {
		name = filepath.Base(filepath.Dir(path))
	}
	description := str(fm.Fields["description"])
	real, err := realPath(path)
	if err != nil {
		return nil, err
	}
	root := filepath.Dir(real)
	if _, ok := cache[root]; !ok {
		hash, err := e.treeDigest(filepath.Dir(path))
		record := M{"hash": hash, "error": nil}
		if err != nil {
			record["hash"] = nil
			record["error"] = err.Error()
		}
		cache[root] = record
	}
	return M{"path": path, "real_path": real, "name": name, "description": description, "tokens_estimate": tokens(fm.Body), "index_tokens_estimate": tokens(name + "\n" + description + "\n" + path), "tree_sha256": cache[root]["hash"], "fingerprint_error": cache[root]["error"], "diagnostics": fm.Problems, "harnesses": []string{}}, nil
}

func (e *Engine) Scan(project string, roots []string) (M, error) {
	project, err := realPath(project)
	if err != nil {
		return nil, err
	}
	scopes, err := projectScopes(project)
	if err != nil {
		return nil, err
	}
	profiles := e.Config.Harnesses
	if roots != nil {
		for i, root := range roots {
			roots[i], err = absolute(root)
			if err != nil {
				return nil, err
			}
		}
		profiles = map[string]Profile{"custom": {SkillRoots: roots, Rules: []RuleSpec{}, Status: "custom"}}
	}
	records, rules := map[string]M{}, map[string]M{}
	cache := map[string]M{}
	problems, coverage := []M{}, []M{}
	for _, harness := range sortedKeys(profiles) {
		profile := profiles[harness]
		for _, pattern := range profile.SkillRoots {
			matches, err := e.scopedPaths(pattern, scopes)
			if err != nil {
				problems = append(problems, M{"path": pattern, "error": err.Error()})
				matches = []string{}
			}
			coverage = append(coverage, M{"harness": harness, "pattern": pattern, "matches": matches, "status": profile.Status})
			for _, root := range matches {
				paths, issues := walkSkills(root, e.Config.Limits["scan_dirs"])
				for _, issue := range issues {
					problems = append(problems, M{"path": root, "error": issue})
				}
				for _, path := range paths {
					if _, ok := records[path]; !ok {
						record, err := e.skillRecord(path, cache)
						if err != nil {
							problems = append(problems, M{"path": path, "error": err.Error()})
							continue
						}
						records[path] = record
						if record["fingerprint_error"] != nil {
							problems = append(problems, M{"path": path, "error": record["fingerprint_error"], "stage": "fingerprint"})
						}
					}
					exposure := stringsOf(records[path]["harnesses"])
					if !contains(exposure, harness) {
						records[path]["harnesses"] = append(exposure, harness)
					}
				}
			}
		}
		for _, spec := range profile.Rules {
			paths, err := e.scopedPaths(spec.Path, scopes)
			if err != nil {
				problems = append(problems, M{"path": spec.Path, "error": err.Error()})
				continue
			}
			for _, path := range paths {
				if !isFile(path) {
					continue
				}
				text, err := readText(path, e.Config.Limits["file_bytes"])
				if err != nil {
					problems = append(problems, M{"path": path, "error": err.Error()})
					continue
				}
				mode := spec.Load
				if mode == "" {
					mode = "candidate"
				}
				if mode == "frontmatter" {
					mode = "conditional"
					if parseFrontmatter(text).Fields["alwaysApply"] == true {
						mode = "always"
					}
				}
				if harness == "codex" {
					if filepath.Base(path) == "AGENTS.override.md" {
						mode = "always"
					}
					if filepath.Base(path) == "AGENTS.md" && isFile(filepath.Join(filepath.Dir(path), "AGENTS.override.md")) {
						mode = "shadowed"
					}
				}
				if rules[path] == nil {
					rules[path] = M{"path": path, "tokens_estimate": tokens(text), "harnesses": map[string]string{}}
				}
				rules[path]["harnesses"].(map[string]string)[harness] = mode
			}
		}
	}
	skills, ruleList := []M{}, []M{}
	groups := map[string][]M{}
	for _, path := range sortedKeys(records) {
		record := records[path]
		skills = append(skills, record)
		groups[str(record["name"])] = append(groups[str(record["name"])], record)
	}
	for _, path := range sortedKeys(rules) {
		ruleList = append(ruleList, rules[path])
	}
	duplicates := []M{}
	for _, name := range sortedKeys(groups) {
		copies := groups[name]
		if len(copies) < 2 {
			continue
		}
		physical, hashes := map[string]bool{}, map[string]bool{}
		byPhysical, byHash := map[string][]M{}, map[string][]M{}
		unverified := false
		paths := []string{}
		exposure := map[string]int{}
		for h := range profiles {
			exposure[h] = 0
		}
		for _, copy := range copies {
			physical[str(copy["real_path"])] = true
			byPhysical[str(copy["real_path"])] = append(byPhysical[str(copy["real_path"])], copy)
			paths = append(paths, str(copy["path"]))
			if copy["tree_sha256"] == nil {
				unverified = true
			} else {
				hashes[str(copy["tree_sha256"])] = true
				byHash[str(copy["tree_sha256"])] = append(byHash[str(copy["tree_sha256"])], copy)
			}
			for _, h := range stringsOf(copy["harnesses"]) {
				exposure[h]++
			}
		}
		kind := "divergent"
		if len(physical) == 1 {
			kind = "aliases"
		} else if unverified {
			kind = "unverified"
		} else if len(hashes) == 1 {
			kind = "identical"
		}
		group := M{"name": name, "kind": kind, "paths": paths, "harness_exposure": exposure}
		// Preserve relationships inside mixed sets without changing simple-pair reports.
		if len(physical) > 1 {
			aliases, identical := []M{}, []M{}
			for _, real := range sortedKeys(byPhysical) {
				if len(byPhysical[real]) > 1 {
					members := []string{}
					for _, copy := range byPhysical[real] {
						members = append(members, str(copy["path"]))
					}
					aliases = append(aliases, M{"real_path": real, "paths": members})
				}
			}
			for _, hash := range sortedKeys(byHash) {
				roots, members := map[string]bool{}, []string{}
				for _, copy := range byHash[hash] {
					roots[str(copy["real_path"])] = true
					members = append(members, str(copy["path"]))
				}
				if len(roots) > 1 {
					identical = append(identical, M{"tree_sha256": hash, "paths": members})
				}
			}
			if len(aliases) > 0 || len(hashes) > 1 && len(identical) > 0 {
				group["alias_groups"], group["identical_groups"] = aliases, identical
			}
		}
		duplicates = append(duplicates, group)
	}
	summaries := map[string]M{}
	for harness := range profiles {
		count, index, always, conditional := 0, 0, 0, 0
		physical := map[string]bool{}
		for _, record := range skills {
			if contains(stringsOf(record["harnesses"]), harness) {
				count++
				index += integer(record["index_tokens_estimate"])
				physical[str(record["real_path"])] = true
			}
		}
		for _, rule := range ruleList {
			switch rule["harnesses"].(map[string]string)[harness] {
			case "always":
				always += integer(rule["tokens_estimate"])
			case "candidate", "conditional":
				conditional += integer(rule["tokens_estimate"])
			}
		}
		summaries[harness] = M{"skill_occurrences": count, "unique_physical_skills": len(physical), "index_tokens_estimate": index, "always_rule_tokens_estimate": always, "conditional_rule_tokens_estimate": conditional, "over_index_budget": index > e.Config.Budgets["index_tokens"], "over_eager_budget": always > e.Config.Budgets["eager_tokens"]}
	}
	return M{"version": Version, "project": project, "project_scopes": scopes, "token_method": "ceil(UTF-8 bytes / 4); heuristic, not exact", "scope": "Configured paths only; exposure estimates, not observed harness prompt contents or precedence", "skills": skills, "rules": ruleList, "duplicates": duplicates, "summaries": summaries, "coverage": coverage, "errors": problems}, nil
}
