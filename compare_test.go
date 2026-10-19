package agentforge

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestCompareExplainsExecutableOnlyDivergence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable bits are not exposed on Windows")
	}
	f := setup(t)
	left := f.skill("left/example", "", "", "")
	right := f.skill("right/example", "", "", "")
	for _, root := range []string{left, right} {
		writeFile(t, filepath.Join(root, "scripts", "run.py"), "print('not executed')\n")
	}
	must(t, os.Chmod(filepath.Join(left, "scripts", "run.py"), 0755))
	result, err := f.e.Compare(left, right, CompareOptions{})
	must(t, err)
	differences := result["differences"].([]M)
	if result["kind"] != "divergent" || result["byte_content_equal"] != true || len(differences) != 1 ||
		differences[0]["path"] != "scripts/run.py" || !contains(stringsOf(differences[0]["fields"]), "executable_bits") ||
		contains(stringsOf(differences[0]["fields"]), "bytes") {
		t.Fatal(result)
	}
	inventory, err := f.e.Scan(f.root, []string{left, right})
	must(t, err)
	group := inventory["duplicates"].([]M)[0]
	if len(group["difference_summaries"].([]M)) != 1 {
		t.Fatal(group)
	}
}

func TestNormalizedHarnessVariantStillCannotBeDeduplicated(t *testing.T) {
	f := setup(t)
	left := f.skill("left/example", "", "", "Run ~/.agents/skills/example/scripts/run.py. Verify output.\n")
	right := f.skill("right/example", "", "", "Run ~/.cursor/skills/example/scripts/run.py. Verify output.\n")
	result, err := f.e.Compare(left, right, CompareOptions{NormalizeHarnessPaths: true, ShowLines: true})
	must(t, err)
	if result["kind"] != "divergent" || result["content_relation"] != "harness-path-variant" || result["cleanup_eligible"] != false {
		t.Fatal(result)
	}
	lines := result["differences"].([]M)[0]["normalized_lines"].(M)
	if integer(lines["added"]) != 0 || integer(lines["removed"]) != 0 {
		t.Fatal(result)
	}
	_, err = f.e.Archive([]string{right}, filepath.Join(f.root, "state"), true, left)
	reject(t, err)
	if !exists(right) {
		t.Fatal("normalization caused deletion")
	}
}

func TestComparisonPreservesFrontmatterAndOtherSkillPathChanges(t *testing.T) {
	f := setup(t)
	left := f.skill("left/example", "", "allowed-tools: Read\n", "Read ~/.agents/skills/other/guide.md. Verify output.\n")
	right := f.skill("right/example", "", "allowed-tools: Bash\n", "Read ~/.cursor/skills/other/guide.md. Verify output.\n")
	result, err := f.e.Compare(left, right, CompareOptions{NormalizeHarnessPaths: true})
	must(t, err)
	difference := result["differences"].([]M)[0]
	if result["content_relation"] != "content-differs" || !contains(stringsOf(difference["frontmatter_fields"]), "allowed-tools") ||
		integer(difference["normalized_lines"].(M)["added"]) == 0 {
		t.Fatal(result)
	}
}

func TestComparisonReportsSubstantiveLinesAndDoesNotExecute(t *testing.T) {
	f := setup(t)
	left := f.skill("left/example", "", "", "Verify the first result.\n")
	right := f.skill("right/example", "", "", "Verify the second result.\n")
	marker := filepath.Join(f.root, "executed")
	for _, root := range []string{left, right} {
		writeFile(t, filepath.Join(root, "scripts", "run.py"), "open("+strconv.Quote(marker)+", 'w').write('executed')\n")
	}
	result, err := f.e.Compare(left, right, CompareOptions{ShowLines: true})
	must(t, err)
	lines := result["differences"].([]M)[0]["raw_lines"].(M)
	if exists(marker) || integer(lines["added"]) != 1 || integer(lines["removed"]) != 1 || len(lines["lines"].([]M)) != 2 {
		t.Fatal(result)
	}
}

func TestComparePresenceTypesAndSymlinkTargets(t *testing.T) {
	f := setup(t)
	left := f.skill("left/example", "", "", "")
	right := f.skill("right/example", "", "", "")
	must(t, os.Mkdir(filepath.Join(left, "empty"), 0700))
	writeFile(t, filepath.Join(right, "empty"), "file, not directory")
	link(t, "first-target", filepath.Join(left, "link"))
	link(t, "second-target", filepath.Join(right, "link"))
	writeFile(t, filepath.Join(right, "added"), "new")
	result, err := f.e.Compare(left, right, CompareOptions{})
	must(t, err)
	causes := []string{}
	for _, difference := range result["differences"].([]M) {
		causes = append(causes, stringsOf(difference["fields"])...)
	}
	for _, expected := range []string{"presence", "type", "symlink_target"} {
		if !contains(causes, expected) {
			t.Fatal(result)
		}
	}
}

func TestCompareGroupExcludesManagedAndDivergentCopies(t *testing.T) {
	f := setup(t)
	canonical := f.skill("copies/a/example", "", "", "")
	copy := f.skill("copies/b/example", "", "", "")
	f.skill("copies/c/example", "", "", "Verify different output.\n")
	managed := f.skill("copies/.codex/plugins/cache/vendor/example", "", "", "")
	code, result, stderr := runCLI(t, "compare", "--canonical", canonical, "--root", filepath.Join(f.root, "copies"), "--copy", copy, "--copy", managed, "--project", f.root)
	if code != 0 || stderr != "" || len(result["cleanup_candidates"].([]any)) != 1 ||
		result["cleanup_candidates"].([]any)[0] != copy || contains(stringsOf(result["dedupe_preview_argv"]), "--apply") {
		t.Fatal(code, result, stderr)
	}
	result, err := f.e.CompareGroup(canonical, nil, []string{filepath.Join(f.root, "copies"), managed}, f.root, CompareOptions{})
	must(t, err)
	if len(result["comparisons"].([]M)) != 3 || len(result["cleanup_candidates"].([]string)) != 1 {
		t.Fatal(result)
	}
	if !exists(managed) || !exists(copy) {
		t.Fatal("read-only comparison changed copies")
	}
}

func TestCompareAliasThroughParentIsNotCleanupCandidate(t *testing.T) {
	f := setup(t)
	canonical := f.skill("store/example", "", "", "")
	parent := filepath.Join(f.root, "parent-alias")
	link(t, filepath.Dir(canonical), parent)
	result, err := f.e.Compare(canonical, filepath.Join(parent, "example"), CompareOptions{})
	must(t, err)
	if result["kind"] != "aliases" || result["cleanup_eligible"] != false {
		t.Fatal(result)
	}
}

func TestLineOutputAndWorkAreBounded(t *testing.T) {
	budget := 5
	result := lineChanges("long old line\n", "long new line\n", true, &budget)
	if result["truncated"] != true || len(result["lines"].([]M)) != 0 {
		t.Fatal(result)
	}
	result = lineChanges(strings.Repeat("line\n", 2000), strings.Repeat("line\n", 2000), false, &budget)
	if result["available"] != false {
		t.Fatal(result)
	}
}

func TestManagedLintIsUpstreamAndSuppressesPruneSuggestions(t *testing.T) {
	f := setup(t)
	line := "Verify that every generated report preserves all supplied facts and records missing information.\n"
	skill := f.skill(".codex/plugins/cache/vendor/wrong-directory", "example", "context: fork\n", line+line)
	result, err := f.e.LintSkill(skill)
	must(t, err)
	if result["managed"] != true || result["cleanup_eligible"] != false || len(result["prune_candidates"].([]M)) != 0 || !hasIssue(result, "name-directory") {
		t.Fatal(result)
	}
	for _, issue := range result["issues"].([]M) {
		if issue["actionable"] != false || issue["actionability"] != "upstream" || str(issue["meaning"]) == "" {
			t.Fatal(issue)
		}
	}
	inventory, err := f.e.Scan(f.root, []string{skill})
	must(t, err)
	if inventory["skills"].([]M)[0]["managed"] != true || inventory["skills"].([]M)[0]["cleanup_eligible"] != false {
		t.Fatal(inventory)
	}
}

func TestComparisonRejectsOversizedPackages(t *testing.T) {
	f := setup(t)
	left := f.skill("left/example", "", "", "")
	right := f.skill("right/example", "", "", "")
	f.e.Config.Limits["hash_file_bytes"] = 8
	_, err := f.e.Compare(left, right, CompareOptions{})
	reject(t, err)
}

func TestComparisonDoesNotSuggestArchivingRunningInstallation(t *testing.T) {
	f := setup(t)
	left := f.skill("left/example", "", "", "")
	right := f.skill("right/example", "", "", "")
	f.e.SkillDir = right
	result, err := f.e.Compare(left, right, CompareOptions{})
	must(t, err)
	if result["kind"] != "identical" || result["cleanup_eligible"] != false {
		t.Fatal(result)
	}
}

func TestManagedLocationThroughSymlinkedParentStaysProtected(t *testing.T) {
	f := setup(t)
	source := f.skill("owned/example", "", "", "")
	cache := filepath.Join(f.root, ".codex", "plugins", "cache", "vendor")
	must(t, os.MkdirAll(filepath.Dir(cache), 0700))
	link(t, filepath.Dir(source), cache)
	alias := filepath.Join(cache, "example")
	code, result, stderr := runCLI(t, "lint", alias, "--project", f.root)
	if code != 0 || stderr != "" || result["skills"].([]any)[0].(map[string]any)["managed"] != true {
		t.Fatal(code, result, stderr)
	}
	_, err := f.e.Archive([]string{alias}, filepath.Join(f.root, "state"), true, "")
	if err == nil || !exists(source) {
		t.Fatal("managed location moved its user-owned target", err)
	}
}
