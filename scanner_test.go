package agentforge

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type fixture struct {
	t    *testing.T
	root string
	e    *Engine
}

func setup(t *testing.T) *fixture {
	t.Helper()
	e, err := NewEngine("")
	if err != nil {
		t.Fatal(err)
	}
	root, err := realPath(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e.Home = root
	return &fixture{t, root, e}
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func reject(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("operation unexpectedly accepted")
	}
}
func writeFile(t *testing.T, path, text string) {
	t.Helper()
	must(t, os.MkdirAll(filepath.Dir(path), 0700))
	must(t, os.WriteFile(path, []byte(text), 0644))
}
func (f *fixture) skill(relative, name, extra, body string) string {
	f.t.Helper()
	path := filepath.Join(f.root, filepath.FromSlash(relative))
	if name == "" {
		name = "example"
	}
	if body == "" {
		body = "Verify the output.\n"
	}
	writeFile(f.t, filepath.Join(path, "SKILL.md"), "---\nname: "+name+"\ndescription: Validate example data when preparing local reports. Not for image generation.\n"+extra+"---\n"+body)
	return path
}
func link(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("OS account cannot create symlinks: %v", err)
		}
		t.Fatal(err)
	}
}
func hasIssue(report M, code string) bool {
	for _, issue := range report["issues"].([]M) {
		if issue["code"] == code {
			return true
		}
	}
	return false
}
func decoded(t *testing.T, value any) M {
	t.Helper()
	data, err := json.Marshal(value)
	must(t, err)
	result := M{}
	must(t, json.Unmarshal(data, &result))
	return result
}
func runCLI(t *testing.T, args ...string) (int, M, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(args, &out, &errOut)
	result := M{}
	if strings.HasPrefix(strings.TrimSpace(out.String()), "{") {
		must(t, json.Unmarshal(out.Bytes(), &result))
	}
	return code, result, errOut.String()
}

func TestFrontmatterFoldedAndOpaque(t *testing.T) {
	fm := parseFrontmatter("---\nname: example\ndescription: >-\n  Read local data\n  when validating reports.\nmetadata:\n  version: '1'\n---\nBody\n")
	if len(fm.Problems) > 0 || fm.Fields["description"] != "Read local data when validating reports." || fm.Body != "Body\n" || !strings.Contains(fm.Blocks["metadata"], "version: '1'") {
		t.Fatalf("%+v", fm)
	}
}
func TestFrontmatterQuotedAndDuplicate(t *testing.T) {
	fm := parseFrontmatter("---\nname: example\ndescription: 'Don''t guess' # comment\nname: other\n---\n")
	if fm.Fields["description"] != "Don't guess" || len(fm.Problems) == 0 {
		t.Fatalf("%+v", fm)
	}
}
func TestFrontmatterRejectsUnsupportedRoutingScalars(t *testing.T) {
	for _, text := range []string{"Body", "---\nname: example", "---\nname: example\ndescription: &anchor test\n---\n", "---\nname: example\ndescription: 'unclosed\n---\n"} {
		if len(parseFrontmatter(text).Problems) == 0 {
			t.Fatalf("accepted %q", text)
		}
	}
}
func TestFingerprintDetectsScriptAndEmptyDirectory(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "")
	first, err := f.e.treeDigest(skill)
	must(t, err)
	must(t, os.Mkdir(filepath.Join(skill, "empty"), 0700))
	second, err := f.e.treeDigest(skill)
	must(t, err)
	if first == second {
		t.Fatal("empty directory did not change fingerprint")
	}
	writeFile(t, filepath.Join(skill, "run.py"), "print(1)")
	third, err := f.e.treeDigest(skill)
	must(t, err)
	if second == third {
		t.Fatal("script did not change fingerprint")
	}
}
func TestFingerprintDetectsExecutableMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX executable mode semantics")
	}
	f := setup(t)
	skill := f.skill("example", "", "", "")
	script := filepath.Join(skill, "run.sh")
	writeFile(t, script, "echo example")
	before, err := f.e.treeDigest(skill)
	must(t, err)
	must(t, os.Chmod(script, 0755))
	after, err := f.e.treeDigest(skill)
	must(t, err)
	if before == after {
		t.Fatal("permission change was lost")
	}
}
func TestFingerprintMissingAndOversize(t *testing.T) {
	f := setup(t)
	_, err := f.e.treeDigest(filepath.Join(f.root, "missing"))
	reject(t, err)
	skill := f.skill("example", "", "", "")
	f.e.Config.Limits["hash_file_bytes"] = 1
	_, err = f.e.treeDigest(skill)
	reject(t, err)
}
func TestInventoryAliasesAndCycles(t *testing.T) {
	f := setup(t)
	skill := f.skill("store/example", "", "", "")
	root := filepath.Join(f.root, "skills")
	must(t, os.Mkdir(root, 0700))
	link(t, skill, filepath.Join(root, "first"))
	link(t, skill, filepath.Join(root, "second"))
	link(t, root, filepath.Join(root, "cycle"))
	result, err := f.e.Scan(f.root, []string{root})
	must(t, err)
	if len(result["errors"].([]M)) != 0 || len(result["skills"].([]M)) != 2 || result["duplicates"].([]M)[0]["kind"] != "aliases" {
		t.Fatalf("%v", result)
	}
}
func TestInventoryDivergentScriptContent(t *testing.T) {
	f := setup(t)
	left := f.skill("a/example", "", "", "")
	right := f.skill("b/example", "", "", "")
	writeFile(t, filepath.Join(right, "script.py"), "print(2)")
	result, err := f.e.Scan(f.root, []string{filepath.Dir(left), filepath.Dir(right)})
	must(t, err)
	if result["duplicates"].([]M)[0]["kind"] != "divergent" {
		t.Fatal(result)
	}
}
func TestEagerRulesSeparateConditional(t *testing.T) {
	f := setup(t)
	rules := filepath.Join(f.root, "rules")
	writeFile(t, filepath.Join(rules, "always.mdc"), "---\nalwaysApply: true\n---\nAlways instructions")
	writeFile(t, filepath.Join(rules, "conditional.mdc"), "---\nalwaysApply: false\n---\nConditional instructions")
	f.e.Config.Harnesses = map[string]Profile{"test": {Rules: []RuleSpec{{filepath.Join(rules, "*.mdc"), "frontmatter"}}}}
	result, err := f.e.Scan(f.root, nil)
	must(t, err)
	summary := result["summaries"].(map[string]M)["test"]
	if integer(summary["always_rule_tokens_estimate"]) == 0 || integer(summary["conditional_rule_tokens_estimate"]) == 0 {
		t.Fatal(summary)
	}
}
func TestInventoryUnreadableMetadata(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "")
	must(t, os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte{0xff}, 0644))
	result, err := f.e.Scan(f.root, []string{skill})
	must(t, err)
	if len(result["errors"].([]M)) != 1 {
		t.Fatal(result)
	}
}
func TestInventoryRetainsIncompleteFingerprint(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "")
	f.e.Config.Limits["hash_file_bytes"] = 16
	result, err := f.e.Scan(f.root, []string{skill})
	must(t, err)
	skills := result["skills"].([]M)
	if len(skills) != 1 || skills[0]["tree_sha256"] != nil || skills[0]["fingerprint_error"] == nil {
		t.Fatal(result)
	}
}
func TestAncestorScanStopsAtWorktree(t *testing.T) {
	f := setup(t)
	repo := filepath.Join(f.root, "repo")
	nested := filepath.Join(repo, "src", "module")
	must(t, os.MkdirAll(nested, 0700))
	must(t, os.Mkdir(filepath.Join(repo, ".git"), 0700))
	f.skill("repo/.agents/skills/example", "", "", "")
	f.e.Config.Harnesses = map[string]Profile{"test": {SkillRoots: []string{"{project}/.agents/skills"}}}
	result, err := f.e.Scan(nested, nil)
	must(t, err)
	scopes := result["project_scopes"].([]string)
	if len(result["skills"].([]M)) != 1 || scopes[len(scopes)-1] != repo || contains(scopes, f.root) {
		t.Fatal(result)
	}
}
func TestCodexOverrideShadowsBase(t *testing.T) {
	f := setup(t)
	writeFile(t, filepath.Join(f.root, "AGENTS.md"), "base")
	writeFile(t, filepath.Join(f.root, "AGENTS.override.md"), "override")
	f.e.Config.Harnesses = map[string]Profile{"codex": {Rules: []RuleSpec{{"{project}/AGENTS.md", "always"}, {"{project}/AGENTS.override.md", "candidate"}}}}
	result, err := f.e.Scan(f.root, nil)
	must(t, err)
	summary := result["summaries"].(map[string]M)["codex"]
	if integer(summary["always_rule_tokens_estimate"]) != tokens("override") || integer(summary["conditional_rule_tokens_estimate"]) != 0 {
		t.Fatal(summary)
	}
}
func TestCodexBuiltinRootDoesNotExposeOtherHiddenFolders(t *testing.T) {
	f := setup(t)
	t.Setenv("CODEX_HOME", filepath.Join(f.root, ".codex"))
	f.skill(".codex/skills/.system/builtin", "builtin", "", "")
	f.skill(".codex/skills/.unrelated/hidden", "hidden", "", "")
	f.skill(".codex/skills/ordinary", "ordinary", "", "")
	f.e.Config.Harnesses = map[string]Profile{"codex": f.e.Config.Harnesses["codex"]}
	result, err := f.e.Scan(f.root, nil)
	must(t, err)
	names := []string{}
	for _, skill := range result["skills"].([]M) {
		names = append(names, str(skill["name"]))
	}
	if !reflect.DeepEqual(unique(names), []string{"builtin", "ordinary"}) {
		t.Fatal(names)
	}
}
func TestStateRejectedForAbsentDiscoveryRoot(t *testing.T) {
	f := setup(t)
	f.e.Config.Harnesses = map[string]Profile{"test": {SkillRoots: []string{"{project}/.agents/skills"}}}
	_, err := f.e.outsideRoots(filepath.Join(f.root, ".agents", "skills", "state"), f.root)
	reject(t, err)
}
func TestRecursiveRuleGlob(t *testing.T) {
	f := setup(t)
	writeFile(t, filepath.Join(f.root, "rules", "a.md"), "A")
	writeFile(t, filepath.Join(f.root, "rules", "nested", "b.md"), "B")
	paths, err := f.e.expand("{project}/rules/**/*.md", f.root)
	must(t, err)
	if len(paths) != 2 {
		t.Fatal(paths)
	}
}
func TestConfigOverlayValidationAndEmbeddedDefaults(t *testing.T) {
	f := setup(t)
	path := filepath.Join(f.root, "config.json")
	writeFile(t, path, `{"budgets":{"index_tokens":123}}`)
	config, err := loadConfig(path)
	must(t, err)
	if config.Budgets["index_tokens"] != 123 || config.Budgets["eager_tokens"] != 2000 {
		t.Fatal(config)
	}
	writeFile(t, path, `{"sdqs_weights":{"structure":-1}}`)
	_, err = loadConfig(path)
	reject(t, err)
	writeFile(t, path, `{"limits":{"file_bytes":0}}`)
	_, err = loadConfig(path)
	reject(t, err)
	for _, content := range []string{`{"limits":null}`, `{"budgets":null}`, `{"harnesses":null}`} {
		writeFile(t, path, content)
		_, err = loadConfig(path)
		reject(t, err)
	}
}
func TestAtomicJSONReplacesAndLeavesNoTemporaryFile(t *testing.T) {
	f := setup(t)
	path := filepath.Join(f.root, "state", "record.json")
	must(t, atomicJSON(path, M{"version": 1}))
	must(t, atomicJSON(path, M{"version": 2}))
	var value M
	must(t, readJSON(path, &value))
	entries, err := os.ReadDir(filepath.Dir(path))
	must(t, err)
	if integer(value["version"]) != 2 || len(entries) != 1 {
		t.Fatal(value, entries)
	}
}
func TestScanMatchesIndependentPythonGolden(t *testing.T) {
	f := setup(t)
	var fixture struct {
		Files       map[string]string `json:"files"`
		Directories []string          `json:"directories"`
	}
	must(t, readJSON("testdata/scan-fixture.json", &fixture))
	for path, text := range fixture.Files {
		writeFile(t, filepath.Join(f.root, filepath.FromSlash(path)), text)
	}
	for _, path := range fixture.Directories {
		must(t, os.MkdirAll(filepath.Join(f.root, filepath.FromSlash(path)), 0700))
	}
	report, err := f.e.Scan(f.root, []string{filepath.Join(f.root, "skills")})
	must(t, err)
	var normalize func(any) any
	normalize = func(value any) any {
		switch v := value.(type) {
		case string:
			return filepath.ToSlash(strings.ReplaceAll(v, f.root, "$ROOT"))
		case []any:
			for i, item := range v {
				v[i] = normalize(item)
			}
			return v
		case map[string]any:
			for key, item := range v {
				v[key] = normalize(item)
			}
			return v
		default:
			return value
		}
	}
	actual := normalize(decoded(t, report)).(map[string]any)
	sum := 0
	for _, value := range actual["skills"].([]any) {
		skill := value.(map[string]any)
		n := tokens(str(skill["name"]) + "\n" + str(skill["description"]) + "\n" + str(skill["path"]))
		skill["index_tokens_estimate"] = float64(n)
		sum += n
	}
	actual["summaries"].(map[string]any)["custom"].(map[string]any)["index_tokens_estimate"] = float64(sum)
	expected := M{}
	must(t, readJSON("testdata/python-scan-golden.json", &expected))
	if !reflect.DeepEqual(actual, expected) {
		a, _ := json.MarshalIndent(actual, "", "  ")
		b, _ := json.MarshalIndent(expected, "", "  ")
		t.Fatalf("Go scan differs from Python golden:\nGo: %s\nPython: %s", a, b)
	}
}
func TestCLIHelpAndOperationalExitCodes(t *testing.T) {
	for _, route := range commandNames() {
		args := append(strings.Fields(route), "--help")
		code, _, stderr := runCLI(t, args...)
		if code != 0 || stderr != "" {
			t.Fatal(route, code, stderr)
		}
	}
	for _, args := range [][]string{{"missing"}, {"audit", "--unknown"}, {"sync", "cold"}, {"lint", "--fail-under", "NaN"}, {"audit", "--config"}} {
		code, _, stderr := runCLI(t, args...)
		if code != 2 || !strings.Contains(stderr, "error") {
			t.Fatal(args, code, stderr)
		}
	}
}
func TestCLIInventoryBudgetAndGlobalFlagPositions(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "")
	config := filepath.Join(f.root, "overlay.json")
	writeFile(t, config, `{"budgets":{"index_tokens":1}}`)
	for _, args := range [][]string{{"--config", config, "audit", "--root", skill, "--project", f.root, "--fail-on-budget"}, {"scan", "--config=" + config, "--project=" + f.root, "--root=" + skill, "--fail-on-budget"}} {
		code, result, stderr := runCLI(t, args...)
		if code != 1 || stderr != "" || len(result["skills"].([]any)) != 1 {
			t.Fatal(code, result, stderr)
		}
	}
}

func TestCLISyncAndInspectionRoutes(t *testing.T) {
	f := setup(t)
	source := f.skill("example", "", "", "")
	state := filepath.Join(f.root, "state")
	code, result, stderr := runCLI(t, "sync", "cold", "--skill", source, "--state-dir", state, "--project", f.root)
	if code != 0 || stderr != "" || result["applied"] != false || exists(state) {
		t.Fatal(code, result, stderr)
	}
	code, result, stderr = runCLI(t, "sync", "cold", "--skill", source, "--state-dir", state, "--project", f.root, "--apply")
	if code != 0 || stderr != "" || exists(source) {
		t.Fatal(code, result, stderr)
	}
	code, result, stderr = runCLI(t, "sync", "restore", "--transaction", str(result["id"]), "--state-dir", state, "--project", f.root, "--apply")
	if code != 0 || stderr != "" || !exists(source) {
		t.Fatal(code, result, stderr)
	}
	code, result, stderr = runCLI(t, "discover", "inspect", "--path", source)
	if code != 0 || stderr != "" || result["executed"] != false {
		t.Fatal(code, result, stderr)
	}
	code, result, stderr = runCLI(t, "sync", "export", "--skill", source, "--harness", "portable", "--destination", filepath.Join(f.root, "target", "example"), "--state-dir", state, "--project", f.root)
	if code != 0 || stderr != "" || result["applied"] != false {
		t.Fatal(code, result, stderr)
	}
}
func TestLimitsRejectInvalidUTF8AndLargeReads(t *testing.T) {
	f := setup(t)
	path := filepath.Join(f.root, "file")
	writeFile(t, path, "large")
	_, err := readText(path, 2)
	reject(t, err)
	must(t, os.WriteFile(path, []byte{0xff}, 0600))
	_, err = readText(path, 10)
	reject(t, err)
}
