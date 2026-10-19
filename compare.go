package agentforge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

type comparisonNode struct {
	Kind, Hash, Link, Text string
	Executable             uint32
	IsText                 bool
}

type CompareOptions struct {
	NormalizeHarnessPaths bool
	ShowLines             bool
	SummaryOnly           bool
}

// Read symlink targets as metadata; never follow links inside a package.
func (e *Engine) comparisonTree(root string) (map[string]comparisonNode, error) {
	nodes := map[string]comparisonNode{}
	total := int64(0)
	textBytes := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if ignored[entry.Name()] && isDir(path) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if len(nodes) >= e.Config.Limits["tree_files"] {
			return errors.New("comparison entry limit exceeded")
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		node := comparisonNode{Kind: "directory"}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			node.Kind = "symlink"
			node.Link, err = os.Readlink(path)
			if err != nil {
				return err
			}
			total += int64(len(node.Link) + 1)
		case info.Mode().IsRegular():
			if info.Size() > int64(e.Config.Limits["hash_file_bytes"]) {
				return fmt.Errorf("file exceeds comparison limit: %s", relative)
			}
			total += info.Size() + int64(len(executableHeader(info.Mode())))
			if total > int64(e.Config.Limits["tree_bytes"]) {
				return errors.New("comparison byte limit exceeded")
			}
			node.Kind, node.Executable = "file", uint32(info.Mode().Perm()&0111)
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			hash := sha256.New()
			var content bytes.Buffer
			writer := io.Writer(hash)
			if info.Size() <= int64(e.Config.Limits["file_bytes"]) && info.Size() <= int64(8_388_608-textBytes) {
				writer = io.MultiWriter(hash, &content)
			}
			read, readErr := io.Copy(writer, io.LimitReader(file, info.Size()+1))
			closeErr := file.Close()
			if readErr != nil {
				return readErr
			}
			if closeErr != nil {
				return closeErr
			}
			if read != info.Size() {
				return fmt.Errorf("file changed during comparison: %s", relative)
			}
			node.Hash = hex.EncodeToString(hash.Sum(nil))
			if int64(content.Len()) == info.Size() && utf8.Valid(content.Bytes()) && !bytes.ContainsRune(content.Bytes(), 0) {
				node.Text, node.IsText = content.String(), true
				textBytes += content.Len()
			}
		case !info.IsDir():
			return fmt.Errorf("unsupported comparison entry: %s", relative)
		}
		if total > int64(e.Config.Limits["tree_bytes"]) {
			return errors.New("comparison byte limit exceeded")
		}
		nodes[relative] = node
		return nil
	})
	return nodes, err
}

func normalizeSkillPaths(text, root, name string) string {
	boundary := `($|[^A-Za-z0-9_.-])`
	for _, spelling := range unique([]string{root, filepath.ToSlash(root)}) {
		text = regexp.MustCompile(regexp.QuoteMeta(spelling)+boundary).ReplaceAllString(text, "{skill}$1")
	}
	if validName(name) {
		roots := `(?:~[/\\])?(?:\.(?:agents|agent|claude|cursor|codex|devin|cognition|windsurf|gemini|opencode|hermes)[/\\]skills|\.config[/\\](?:devin|opencode)[/\\]skills)[/\\]`
		text = regexp.MustCompile(roots+regexp.QuoteMeta(name)+boundary).ReplaceAllString(text, "{skill}$1")
	}
	return text
}

func changedFrontmatter(left, right string) []string {
	a, b := parseFrontmatter(left), parseFrontmatter(right)
	fields := append(sortedKeys(a.Blocks), sortedKeys(b.Blocks)...)
	changed := []string{}
	for _, key := range unique(fields) {
		if a.Blocks[key] != b.Blocks[key] {
			changed = append(changed, key)
		}
	}
	return changed
}

// Bound the LCS table so a large generated file cannot exhaust memory.
func lineChanges(left, right string, show bool, budget *int) M {
	a, b := strings.Split(left, "\n"), strings.Split(right, "\n")
	if (len(a) + 1) > 2_000_000/(len(b)+1) {
		return M{"available": false, "reason": "line comparison limit exceeded"}
	}
	width := len(b) + 1
	table := make([]uint32, (len(a)+1)*width)
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				table[i*width+j] = table[(i+1)*width+j+1] + 1
			} else {
				table[i*width+j] = max(table[(i+1)*width+j], table[i*width+j+1])
			}
		}
	}
	added, removed := len(b)-int(table[0]), len(a)-int(table[0])
	result := M{"available": true, "added": added, "removed": removed}
	if !show {
		return result
	}
	lines := []M{}
	truncated := false
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		if i < len(a) && j < len(b) && a[i] == b[j] {
			i++
			j++
			continue
		}
		side, number, text := "copy", j+1, ""
		if i < len(a) && (j == len(b) || table[(i+1)*width+j] >= table[i*width+j+1]) {
			side, number, text = "canonical", i+1, a[i]
			i++
		} else {
			text = b[j]
			j++
		}
		if len(lines) >= 100 || len(text) > *budget {
			truncated = true
			continue
		}
		lines = append(lines, M{"side": side, "line": number, "text": text})
		*budget -= len(text) + 1
	}
	result["lines"], result["truncated"] = lines, truncated
	return result
}

func comparisonRoot(path string) (string, error) {
	path, err := absolute(path)
	if err != nil {
		return "", err
	}
	if filepath.Base(path) == "SKILL.md" {
		path = filepath.Dir(path)
	}
	if !isDir(path) || !isFile(filepath.Join(path, "SKILL.md")) {
		return "", fmt.Errorf("expected a skill with SKILL.md: %s", path)
	}
	return path, nil
}

func (e *Engine) Compare(canonical, copy string, options CompareOptions) (M, error) {
	normalize, showLines := options.NormalizeHarnessPaths, options.ShowLines
	canonical, err := comparisonRoot(canonical)
	if err != nil {
		return nil, err
	}
	copy, err = comparisonRoot(copy)
	if err != nil {
		return nil, err
	}
	beforeA, err := e.treeDigest(canonical)
	if err != nil {
		return nil, err
	}
	beforeB, err := e.treeDigest(copy)
	if err != nil {
		return nil, err
	}
	realA, err := realPath(canonical)
	if err != nil {
		return nil, err
	}
	realB, err := realPath(copy)
	if err != nil {
		return nil, err
	}
	a, err := e.comparisonTree(realA)
	if err != nil {
		return nil, err
	}
	b, err := e.comparisonTree(realB)
	if err != nil {
		return nil, err
	}
	nameA := str(parseFrontmatter(a["SKILL.md"].Text).Fields["name"])
	nameB := str(parseFrontmatter(b["SKILL.md"].Text).Fields["name"])
	paths := unique(append(sortedKeys(a), sortedKeys(b)...))
	differences := []M{}
	count, contentEqual, normalizedEqual, budget := 0, true, true, 16_384
	for _, path := range paths {
		left, leftOK := a[path]
		right, rightOK := b[path]
		fields := []string{}
		difference := M{"path": path}
		switch {
		case !leftOK || !rightOK:
			fields = append(fields, "presence")
			difference["canonical_present"], difference["copy_present"] = leftOK, rightOK
			contentEqual, normalizedEqual = false, false
		case left.Kind != right.Kind:
			fields = append(fields, "type")
			difference["canonical_type"], difference["copy_type"] = left.Kind, right.Kind
			contentEqual, normalizedEqual = false, false
		case left.Kind == "symlink" && left.Link != right.Link:
			fields = append(fields, "symlink_target")
			difference["canonical_target"], difference["copy_target"] = left.Link, right.Link
			contentEqual, normalizedEqual = false, false
		case left.Kind == "file":
			if left.Executable != right.Executable {
				fields = append(fields, "executable_bits")
				difference["canonical_executable_bits"] = fmt.Sprintf("%03o", left.Executable)
				difference["copy_executable_bits"] = fmt.Sprintf("%03o", right.Executable)
			}
			if left.Hash != right.Hash {
				fields = append(fields, "bytes")
				contentEqual = false
				normalA, normalB := left.Text, right.Text
				if normalize && left.IsText && right.IsText {
					normalA = normalizeSkillPaths(left.Text, canonical, nameA)
					normalB = normalizeSkillPaths(right.Text, copy, nameB)
				}
				equal := normalize && left.IsText && right.IsText && normalA == normalB
				if !equal {
					normalizedEqual = false
				}
				difference["harness_paths_only"] = equal
				if path == "SKILL.md" && left.IsText && right.IsText {
					difference["frontmatter_fields"] = changedFrontmatter(left.Text, right.Text)
				}
				if !options.SummaryOnly && left.IsText && right.IsText && len(differences) < 200 {
					difference["raw_lines"] = lineChanges(left.Text, right.Text, showLines && !normalize, &budget)
					if normalize {
						difference["normalized_lines"] = lineChanges(normalA, normalB, showLines, &budget)
					}
				}
			}
		}
		if len(fields) > 0 {
			count++
			difference["fields"] = fields
			if len(differences) < 200 {
				differences = append(differences, difference)
			}
		}
	}
	afterA, err := e.treeDigest(canonical)
	if err != nil {
		return nil, err
	}
	afterB, err := e.treeDigest(copy)
	if err != nil {
		return nil, err
	}
	if beforeA != afterA || beforeB != afterB {
		return nil, errors.New("package changed during comparison; retry against stable copies")
	}
	kind := "divergent"
	if realA == realB {
		kind = "aliases"
	} else if beforeA == beforeB {
		kind = "identical"
	}
	relation := "content-differs"
	if contentEqual {
		relation = "same-bytes"
	} else if normalize && normalizedEqual {
		relation = "harness-path-variant"
	}
	managed := managedReason(copy)
	canonicalFM, copyFM := parseFrontmatter(a["SKILL.md"].Text), parseFrontmatter(b["SKILL.md"].Text)
	metadataValid := func(fm Frontmatter) bool {
		length := len([]rune(str(fm.Fields["description"])))
		return len(fm.Problems) == 0 && validName(str(fm.Fields["name"])) && length > 0 && length <= 1024
	}
	eligible := kind != "divergent" && managed == "" && managedReason(canonical) == "" &&
		nameA == nameB && metadataValid(canonicalFM) && metadataValid(copyFM)
	if kind == "aliases" && !isLink(copy) {
		eligible = false
	}
	if kind == "identical" {
		for _, node := range a {
			if node.Kind == "symlink" {
				eligible = false
			}
		}
	}
	return M{
		"canonical": canonical, "copy": copy, "kind": kind,
		"canonical_sha256": beforeA, "copy_sha256": beforeB,
		"byte_content_equal": contentEqual, "content_relation": relation,
		"normalization_applied": normalize, "differences": differences,
		"differing_entries": count, "differences_truncated": count > len(differences),
		"managed": managed != "", "managed_reason": managed,
		"cleanup_eligible": eligible,
		"scope":            "Read-only diagnostics. Normalized text is never proof for deduplication.",
	}, nil
}

func (e *Engine) CompareGroup(canonical string, copies, roots []string, project string, options CompareOptions) (M, error) {
	canonical, err := comparisonRoot(canonical)
	if err != nil {
		return nil, err
	}
	errors := []M{}
	if len(copies) == 0 {
		inventory, err := e.Scan(project, roots)
		if err != nil {
			return nil, err
		}
		errors = inventory["errors"].([]M)
		text, err := readText(filepath.Join(canonical, "SKILL.md"), e.Config.Limits["file_bytes"])
		if err != nil {
			return nil, err
		}
		name := str(parseFrontmatter(text).Fields["name"])
		for _, skill := range inventory["skills"].([]M) {
			if skill["name"] == name {
				copies = append(copies, filepath.Dir(str(skill["path"])))
			}
		}
	}
	results, candidates := []M{}, []string{}
	selected := []string{}
	for _, copy := range copies {
		path, err := comparisonRoot(copy)
		if err != nil {
			return nil, err
		}
		selected = append(selected, path)
	}
	for _, copy := range unique(selected) {
		if copy == canonical {
			continue
		}
		result, err := e.Compare(canonical, copy, options)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
		if result["cleanup_eligible"] == true {
			candidates = append(candidates, copy)
		}
	}
	sort.Strings(candidates)
	argv := []string{}
	if len(candidates) > 0 {
		argv = []string{"sync", "dedupe", "--canonical", canonical}
		for _, path := range candidates {
			argv = append(argv, "--duplicate", path)
		}
	}
	return M{"canonical": canonical, "canonical_managed": managedReason(canonical) != "", "comparisons": results,
		"cleanup_candidates": candidates, "dedupe_preview_argv": argv, "errors": errors, "applied": false}, nil
}
