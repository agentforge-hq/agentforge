package agentforge

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const Version = "0.1.0"

//go:embed config.default.json
var defaultConfig []byte

type M = map[string]any

type RuleSpec struct {
	Path string `json:"path"`
	Load string `json:"load"`
}

type Profile struct {
	Status      string     `json:"status"`
	SkillRoots  []string   `json:"skill_roots"`
	Rules       []RuleSpec `json:"rules"`
	Frontmatter []string   `json:"frontmatter"`
	Isolation   *string    `json:"isolation"`
	Source      string     `json:"source"`
}

type Config struct {
	Version            string             `json:"version"`
	Budgets            map[string]int     `json:"budgets"`
	Limits             map[string]int     `json:"limits"`
	Weights            map[string]float64 `json:"sdqs_weights"`
	CollisionThreshold float64            `json:"collision_threshold"`
	RoutingTests       struct {
		MinCosine      float64 `json:"min_cosine"`
		MinSharedTerms int     `json:"min_shared_terms"`
	} `json:"routing_tests"`
	Harnesses     map[string]Profile `json:"harnesses"`
	ManualTargets []string           `json:"manual_targets"`
}

type Engine struct {
	Config   Config
	Home     string
	SkillDir string
	Client   *http.Client
	Rename   func(string, string) error
}

func NewEngine(configPath string) (*Engine, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	e := &Engine{Config: config, Home: home, Rename: os.Rename}
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(filepath.Dir(exe)), filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}
	candidates = append(candidates, filepath.Join(home, ".agents", "skills", "agentforge"))
	for _, candidate := range candidates {
		if text, err := readText(filepath.Join(candidate, "SKILL.md"), 1<<20); err == nil {
			if str(parseFrontmatter(text).Fields["name"]) == "agentforge" {
				e.SkillDir, _ = realPath(candidate)
				break
			}
		}
	}
	return e, nil
}

func loadConfig(path string) (Config, error) {
	base := M{}
	if err := json.Unmarshal(defaultConfig, &base); err != nil {
		return Config{}, err
	}
	if path != "" {
		data, err := readText(path, 1<<20)
		if err != nil {
			return Config{}, err
		}
		patch := M{}
		if err = json.Unmarshal([]byte(data), &patch); err != nil {
			return Config{}, err
		}
		mergeMap(base, patch)
	}
	data, _ := json.Marshal(base)
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	sum := 0.0
	for _, weight := range config.Weights {
		if weight < 0 {
			return config, errors.New("SDQS weights must be nonnegative and sum to 100")
		}
		sum += weight
	}
	if sum != 100 {
		return config, errors.New("SDQS weights must sum to 100")
	}
	for _, group := range []map[string]int{config.Budgets, config.Limits} {
		if len(group) == 0 {
			return config, errors.New("budgets and limits must not be null or empty")
		}
		for _, n := range group {
			if n <= 0 {
				return config, errors.New("budgets and limits must be positive integers")
			}
		}
	}
	for _, name := range []string{"index_tokens", "eager_tokens", "skill_tokens", "router_tokens", "description_tokens"} {
		if config.Budgets[name] <= 0 {
			return config, fmt.Errorf("missing positive budget: %s", name)
		}
	}
	for _, name := range []string{"file_bytes", "hash_file_bytes", "tree_bytes", "extract_bytes", "tree_files", "scan_dirs", "download_bytes"} {
		if config.Limits[name] <= 0 {
			return config, fmt.Errorf("missing positive limit: %s", name)
		}
	}
	if config.Harnesses == nil {
		return config, errors.New("harnesses must be an object, not null")
	}
	if config.RoutingTests.MinCosine < 0 || config.RoutingTests.MinCosine > 1 || config.RoutingTests.MinSharedTerms < 1 {
		return config, errors.New("invalid routing test abstention settings")
	}
	return config, nil
}

func mergeMap(base, patch M) {
	for key, value := range patch {
		nested, ok := value.(map[string]any)
		old, prior := base[key].(map[string]any)
		if ok && prior {
			mergeMap(old, nested)
		} else {
			base[key] = value
		}
	}
}

func str(value any) string { s, _ := value.(string); return s }
func integer(value any) int {
	switch n := value.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}
func stringsOf(value any) []string {
	if items, ok := value.([]string); ok {
		return items
	}
	result := []string{}
	if items, ok := value.([]any); ok {
		for _, item := range items {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
	}
	return result
}
func contains(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func unique(values []string) []string {
	m := map[string]bool{}
	for _, s := range values {
		m[s] = true
	}
	return sortedKeys(m)
}
func tokens(text string) int { return (len([]byte(text)) + 3) / 4 }

var nameRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func validName(name string) bool   { return len(name) > 0 && len(name) <= 64 && nameRE.MatchString(name) }
func hashBytes(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func randomID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}
func exists(path string) bool { _, err := os.Lstat(path); return err == nil }
func isDir(path string) bool  { info, err := os.Stat(path); return err == nil && info.IsDir() }
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
func isLink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}
func within(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
func absolute(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return filepath.Abs(path)
}
func realPath(path string) (string, error) {
	path, err := absolute(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if filepath.Dir(path) == path {
		return path, nil
	}
	parent, err := realPath(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}
func readText(path string, limit int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return "", err
	}
	if len(data) > limit {
		return "", fmt.Errorf("file exceeds %d bytes: %s", limit, path)
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if !utf8.Valid(data) {
		return "", fmt.Errorf("non-UTF-8 text: %s", path)
	}
	return string(data), nil
}
func atomicJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".af-")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func readJSON(path string, value any) error {
	text, err := readText(path, 1<<20)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(text), value)
}
func copyFile(source, destination string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	_, err = io.Copy(output, input)
	closeErr := output.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chmod(destination, mode.Perm())
}
func writeNew(path string, data []byte, mode fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	closeErr := file.Close()
	if err != nil {
		os.Remove(path)
		return err
	}
	return closeErr
}
func (e *Engine) defaultState() string { return filepath.Join(e.Home, ".local", "share", "agentforge") }
func (e *Engine) render(pattern, project string) string {
	codex := os.Getenv("CODEX_HOME")
	if codex == "" {
		codex = filepath.Join(e.Home, ".codex")
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(e.Home, ".config")
	}
	devin := filepath.Join(e.Home, ".config", "devin")
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(e.Home, "AppData", "Roaming")
		}
		devin = filepath.Join(appData, "devin")
	}
	r := strings.NewReplacer("{home}", e.Home, "{project}", project, "{codex_home}", codex, "{xdg_config_home}", xdg, "{devin_config_home}", devin)
	result := filepath.FromSlash(r.Replace(pattern))
	if strings.HasPrefix(result, "~"+string(filepath.Separator)) {
		result = filepath.Join(e.Home, result[2:])
	}
	return result
}

func projectScopes(project string) ([]string, error) {
	project, err := realPath(project)
	if err != nil {
		return nil, err
	}
	if !isDir(project) {
		return nil, fmt.Errorf("project directory does not exist: %s", project)
	}
	chain := []string{}
	for p := project; ; p = filepath.Dir(p) {
		chain = append(chain, p)
		if exists(filepath.Join(p, ".git")) {
			return chain, nil
		}
		if filepath.Dir(p) == p {
			return []string{project}, nil
		}
	}
}

func (e *Engine) expand(pattern, project string) ([]string, error) {
	pattern = e.render(pattern, project)
	if !strings.Contains(pattern, "**") {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		if paths == nil {
			paths = []string{}
		}
		return paths, nil
	}
	volume := filepath.VolumeName(pattern)
	rest := strings.TrimPrefix(pattern, volume)
	start := volume
	if strings.HasPrefix(rest, string(filepath.Separator)) {
		start += string(filepath.Separator)
		rest = strings.TrimPrefix(rest, string(filepath.Separator))
	}
	if start == "" {
		start = "."
	}
	parts := strings.Split(rest, string(filepath.Separator))
	found := []string{}
	visited := 0
	var walk func(string, int) error
	walk = func(base string, i int) error {
		visited++
		if visited > e.Config.Limits["scan_dirs"] {
			return errors.New("glob directory limit reached")
		}
		if i == len(parts) {
			if exists(base) {
				a, err := absolute(base)
				if err != nil {
					return err
				}
				found = append(found, a)
			}
			return nil
		}
		if parts[i] == "**" {
			if err := walk(base, i+1); err != nil {
				return err
			}
			entries, err := os.ReadDir(base)
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if entry.IsDir() {
					if err := walk(filepath.Join(base, entry.Name()), i); err != nil {
						return err
					}
				}
			}
			return nil
		}
		matches, err := filepath.Glob(filepath.Join(base, parts[i]))
		if err != nil {
			return err
		}
		for _, match := range matches {
			if err := walk(match, i+1); err != nil {
				return err
			}
		}
		return nil
	}
	err := walk(start, 0)
	return unique(found), err
}
func (e *Engine) scopedPaths(pattern string, scopes []string) ([]string, error) {
	locations := scopes
	if !strings.Contains(pattern, "{project}") {
		locations = scopes[:1]
	}
	result := []string{}
	for _, project := range locations {
		paths, err := e.expand(pattern, project)
		if err != nil {
			return nil, err
		}
		result = append(result, paths...)
	}
	return unique(result), nil
}

// Context deadlines bound explicit network/Git operations; inspected scripts never run.
const operationTimeout = 20 * time.Second

var ignored = map[string]bool{".git": true, "__pycache__": true, ".pytest_cache": true, ".ruff_cache": true, ".DS_Store": true}

// Keep decimal executable-bit encoding compatible with v0.1 Python transaction fingerprints.
func executableHeader(mode fs.FileMode) []byte {
	return []byte("F" + strconv.Itoa(int(mode.Perm()&0111)) + "\x00")
}
