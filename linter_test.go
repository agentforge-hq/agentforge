package agentforge

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestLiteralPythonFlagsWithoutExecution(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "Run \x60python scripts/run.py --wrong\x60. Verify output.")
	marker := filepath.Join(f.root, "executed")
	writeFile(t, filepath.Join(skill, "scripts", "run.py"), "from pathlib import Path\nPath("+strconv.Quote(marker)+").touch()\nimport argparse\np=argparse.ArgumentParser()\np.add_argument('--right')\n")
	result, err := f.e.LintSkill(skill)
	must(t, err)
	if exists(marker) || !hasIssue(result, "script-flag") || !hasIssue(result, "python-verification-boundary") {
		t.Fatal(result)
	}
}
func TestPythonLexerIgnoresCommentsAndDocstringDeclarations(t *testing.T) {
	result, err := inspectPython("# p.add_argument('--comment')\n\"\"\"p.add_argument('--docstring')\"\"\"\np.add_argument('--real')\n")
	must(t, err)
	flags := stringsOf(result["flags"])
	if !contains(flags, "--real") || contains(flags, "--comment") || contains(flags, "--docstring") {
		t.Fatal(result)
	}
}
func TestPythonDynamicFlagsAreUnverified(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "Run \x60python scripts/run.py --dynamic\x60. Verify output.")
	writeFile(t, filepath.Join(skill, "scripts", "run.py"), "p.add_argument(option)")
	result, err := f.e.LintSkill(skill)
	must(t, err)
	found := false
	for _, issue := range result["issues"].([]M) {
		if issue["code"] == "script-flag" {
			found = issue["severity"] == "warning"
		}
	}
	if !found {
		t.Fatal(result)
	}
}
func TestPythonUnclosedTokensAreReported(t *testing.T) {
	for _, text := range []string{"def broken(", "p.add_argument('unfinished", "value = [)"} {
		_, err := inspectPython(text)
		reject(t, err)
	}
}
func TestGoASTFlagsAndSyntax(t *testing.T) {
	result, err := inspectGo("run.go", "package main\nfunc main(){ flags.String(\"output\", \"\", \"path\"); flags.BoolVar(&value,\"verbose\",false,\"help\") }\n")
	must(t, err)
	if !contains(stringsOf(result["flags"]), "--output") || !contains(stringsOf(result["flags"]), "-output") || !contains(stringsOf(result["flags"]), "--verbose") {
		t.Fatal(result)
	}
	_, err = inspectGo("broken.go", "package main\nfunc broken(")
	reject(t, err)
}
func TestGoDynamicFlagIsNotInvented(t *testing.T) {
	result, err := inspectGo("run.go", "package main\nfunc main(){ flags.String(option,\"\",\"\") }\n")
	must(t, err)
	if result["dynamic_options"] != true {
		t.Fatal(result)
	}
}
func TestRealMissingReferenceAndScriptSyntaxRemainErrors(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "Read \x60references/missing.md\x60. Verify output.")
	writeFile(t, filepath.Join(skill, "scripts", "broken.go"), "package main\nfunc broken(")
	result, err := f.e.LintSkill(skill)
	must(t, err)
	if !hasIssue(result, "missing-reference") || !hasIssue(result, "go-syntax") {
		t.Fatal(result)
	}
}
func TestUnverifiedCommandReportedOncePerInvocation(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "Run \x60go run scripts/broken.go --one --two\x60. Verify output.")
	writeFile(t, filepath.Join(skill, "scripts", "broken.go"), "package main\nfunc broken(")
	result, err := f.e.LintSkill(skill)
	must(t, err)
	count := 0
	for _, issue := range result["issues"].([]M) {
		if issue["code"] == "unverified-command" {
			count++
		}
	}
	if count != 1 || !hasIssue(result, "go-syntax") {
		t.Fatal(result)
	}
}
func TestExamplesAndGeneratedOutputsAreNotDependencies(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "Examples: \x60assets/logo.png\x60, \x60scripts/rotate_pdf.py\x60.\nVerify generated \x60assets/preview.png\x60.\n\x60\x60\x60markdown\nRead [example](references/example.md).\n\x60\x60\x60\nRead [guide](references/required.md).\nVerify output.")
	result, err := f.e.LintSkill(skill)
	must(t, err)
	missing := []string{}
	for _, issue := range result["issues"].([]M) {
		if issue["code"] == "missing-reference" {
			missing = append(missing, str(issue["message"]))
		}
	}
	if len(missing) != 1 || missing[0] != "Missing file: references/required.md" || !contains(stringsOf(result["unverified_references"]), "assets/preview.png") {
		t.Fatal(result)
	}
}
func TestExecutableCommandDependenciesAndInterpreterFlags(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "\x60\x60\x60sh\npython3.11 scripts/missing.py --help\npython3.11 -S scripts/run.py --wrong\n\x60\x60\x60\nVerify output.")
	writeFile(t, filepath.Join(skill, "scripts", "run.py"), "p.add_argument('--right')")
	result, err := f.e.LintSkill(skill)
	must(t, err)
	if !hasIssue(result, "missing-reference") || !hasIssue(result, "script-flag") {
		t.Fatal(result)
	}
}
func TestEscapingReferencesAndScriptsAreRejected(t *testing.T) {
	f := setup(t)
	skill := f.skill("example", "", "", "Read [guide](references/../../outside.md). Verify output.")
	writeFile(t, filepath.Join(f.root, "outside.md"), "external")
	writeFile(t, filepath.Join(f.root, "outside.py"), "raise RuntimeError('not executed')")
	must(t, os.Mkdir(filepath.Join(skill, "scripts"), 0700))
	link(t, filepath.Join(f.root, "outside.py"), filepath.Join(skill, "scripts", "escape.py"))
	result, err := f.e.LintSkill(skill)
	must(t, err)
	if !hasIssue(result, "escaping-reference") || !hasIssue(result, "escaping-script") {
		t.Fatal(result)
	}
}
func TestRoutingCollisionAndAmbiguousTie(t *testing.T) {
	skills := []M{{"name": "one", "description": "Extract document tables", "path": "one"}, {"name": "two", "description": "Extract document tables", "path": "two"}}
	result, err := routingReport(skills, 0.55, 0.1, 2, []TriggerCase{{"document tables", []string{"one"}}}, false)
	must(t, err)
	if len(result["collisions"].([]M)) != 1 || result["trigger_tests"].([]M)[0]["passed"] != false {
		t.Fatal(result)
	}
}
func TestRoutingPositiveAndNoMatch(t *testing.T) {
	skills := []M{{"name": "docs", "description": "Extract document tables", "path": "docs"}, {"name": "images", "description": "Generate colorful images", "path": "images"}}
	result, err := routingReport(skills, 0.55, 0.1, 2, []TriggerCase{{"document tables", []string{"docs"}}, {"forecast weather", []string{}}}, false)
	must(t, err)
	for _, test := range result["trigger_tests"].([]M) {
		if test["passed"] != true {
			t.Fatal(test)
		}
	}
}
func TestRoutingAbstainsOnIncidentalWords(t *testing.T) {
	skills := []M{{"name": "meetings", "description": "Turn notes into meeting agendas", "path": "meetings"}, {"name": "visuals", "description": "Show what changes across scenarios", "path": "visuals"}}
	result, err := routingReport(skills, 0.55, 0.1, 2, []TriggerCase{{"Translate this sentence into French.", []string{}}, {"What is 17 multiplied by 23?", []string{}}, {"Prepare meeting agendas from notes.", []string{"meetings"}}}, false)
	must(t, err)
	for _, test := range result["trigger_tests"].([]M) {
		if test["passed"] != true {
			t.Fatal(test)
		}
	}
}
func TestUnknownQueryWordsReduceEvidence(t *testing.T) {
	skills := []M{{"name": "docs", "description": "Extract document tables", "path": "docs"}}
	result, err := routingReport(skills, 0.55, 0.5, 2, []TriggerCase{{"document tables", []string{"docs"}}, {"document tables volcano guitar nebula penguin bicycle", []string{}}}, false)
	must(t, err)
	for _, test := range result["trigger_tests"].([]M) {
		if test["passed"] != true {
			t.Fatal(test)
		}
	}
}
func TestRoutingAbstentionValidationAndSingleTermOptIn(t *testing.T) {
	for _, minimum := range []float64{-0.1, 1.1, math.NaN()} {
		_, err := routingReport(nil, 0.55, minimum, 2, nil, false)
		reject(t, err)
	}
	_, err := routingReport(nil, 0.55, 0.1, 0, nil, false)
	reject(t, err)
	skills := []M{{"name": "docs", "description": "Extract document tables", "path": "docs"}}
	result, err := routingReport(skills, 0.55, 0.1, 2, []TriggerCase{{"document", []string{}}}, false)
	must(t, err)
	if result["trigger_tests"].([]M)[0]["passed"] != true {
		t.Fatal(result)
	}
	result, err = routingReport(skills, 0.55, 0.1, 1, []TriggerCase{{"document", []string{"docs"}}}, false)
	must(t, err)
	if result["trigger_tests"].([]M)[0]["passed"] != true {
		t.Fatal(result)
	}
}
func TestFullCollisionMatrix(t *testing.T) {
	result, err := routingReport([]M{{"name": "one", "description": "Extract table", "path": "/one"}, {"name": "two", "description": "Generate image", "path": "/two"}}, 0.55, 0.1, 2, nil, true)
	must(t, err)
	matrix := result["matrix"].(M)["cosine"].([][]float64)
	if len(matrix) != 2 || matrix[0][0] != 1 || matrix[0][1] != 0 || matrix[1][1] != 1 {
		t.Fatal(result)
	}
}
