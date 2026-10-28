package agentforge

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

type optionSpec struct {
	Name, Default, Help string
	Boolean, Repeat     bool
}
type parsedOptions struct {
	Values     map[string][]string
	Positional []string
	Help       bool
}

func (p parsedOptions) value(name string) string {
	values := p.Values[name]
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}
func (p parsedOptions) boolean(name string) bool { return p.value(name) == "true" }
func specs(entries ...optionSpec) []optionSpec   { return entries }
func valueOption(name, help string) optionSpec   { return optionSpec{Name: name, Help: help} }
func boolOption(name, help string) optionSpec {
	return optionSpec{Name: name, Boolean: true, Default: "false", Help: help}
}
func repeatOption(name, help string) optionSpec {
	return optionSpec{Name: name, Repeat: true, Help: help}
}

var routes = map[string][]optionSpec{
	"search":           specs(valueOption("query", "Task description to send to the skill catalog"), valueOption("harness", "Optional harness identifier"), valueOption("endpoint", "Explicit HTTPS search endpoint override"), optionSpec{Name: "limit", Default: "5", Help: "Maximum results (1..20)"}, repeatOption("available-tool", "Explicit available tool name; never a path"), boolOption("no-tools", "Explicitly declare no available tools"), optionSpec{Name: "allow-setup", Boolean: true, Help: "Allow candidates needing setup (true or false)"}, optionSpec{Name: "allow-credentials", Boolean: true, Help: "Allow candidates needing credentials (true or false)"}),
	"audit":            specs(repeatOption("root", "Use only these skill roots instead of configured roots and rules"), boolOption("fail-on-budget", "Exit 1 when a configured context budget is exceeded")),
	"compare":          specs(valueOption("canonical", "Explicit skill to retain as the group reference"), repeatOption("copy", "Copy to compare; otherwise discover the canonical skill's name group"), repeatOption("root", "Custom roots for name-group discovery"), boolOption("normalize-harness-paths", "Normalize known self paths for diagnostics; never enables deduplication"), boolOption("show-lines", "Include bounded changed text lines")),
	"lint":             specs(repeatOption("root", "Inventory these skill roots"), valueOption("collision-threshold", "TF-IDF collision threshold (0..1)"), boolOption("full-matrix", "Include all pairwise cosine values"), valueOption("trigger-tests", "Local JSON trigger cases"), optionSpec{Name: "fail-under", Default: "0", Help: "Structural score floor (0..100)"}),
	"sync dedupe":      specs(valueOption("canonical", "Canonical skill to retain"), repeatOption("duplicate", "Identical copy to archive"), boolOption("apply", "Apply the previewed mutation")),
	"sync cold":        specs(repeatOption("skill", "Skills to archive together in one transaction"), boolOption("apply", "Apply the previewed mutation")),
	"sync restore":     specs(valueOption("transaction", "32-character transaction ID"), boolOption("apply", "Apply the previewed restore")),
	"sync export":      specs(valueOption("skill", "Source skill"), valueOption("harness", "Target harness or portable"), valueOption("destination", "Exact new directory, rule or ZIP path"), boolOption("isolate", "Claude Code context: fork"), boolOption("allow-lossy", "Acknowledge fields listed as dropped"), optionSpec{Name: "format", Default: "skill", Help: "skill or cursor-rule"}, boolOption("apply", "Apply the previewed export")),
	"sync promote":     specs(valueOption("quarantine", "Reviewed quarantine directory"), valueOption("destination", "Exact new skill directory"), valueOption("review-sha256", "Review fingerprint from promotion preview"), boolOption("accept-findings", "Acknowledge reviewed findings and unscanned files"), boolOption("apply", "Apply the reviewed promotion")),
	"discover inspect": specs(valueOption("path", "Skill directory to inspect without execution")),
	"discover prepare": specs(valueOption("result", "Saved search response or selected result JSON"), optionSpec{Name: "index", Default: "1", Help: "One-based search result index (1..20)"}, valueOption("destination", "Exact new user-owned skill directory"), valueOption("harness", "Harness where the skill will be used"), repeatOption("root", "Custom roots for local installed-skill comparison")),
	"discover fetch":   specs(valueOption("ref", "Git ref (resolved to immutable commit)"), valueOption("skill-path", "Exact repository-relative skill path"), valueOption("archive-sha256", "Optional independently trusted archive checksum")),
	"discover search":  specs(valueOption("query", "Public GitHub repository search query")),
	"drift audit":      specs(valueOption("skill", "Skill directory"), valueOption("repo", "Git repository"), optionSpec{Name: "base", Default: "HEAD", Help: "Baseline commit/ref"}, repeatOption("transcript", "Local JSON/JSONL/text transcript"), repeatOption("pr", "Local exported PR JSON")),
	"drift distill":    specs(repeatOption("transcript", "Local transcript evidence"), valueOption("name", "Draft skill name"), valueOption("description", "Draft routing description"), optionSpec{Name: "min-occurrences", Default: "2", Help: "Distinct-message repetition floor"}, boolOption("apply", "Save a review-only draft outside discovery roots")),
}
var globalOptions = specs(valueOption("config", "JSON overlay for embedded defaults"), valueOption("project", "Session directory; defaults to current directory"), valueOption("state-dir", "Cold storage/quarantine root outside discovery roots"))

func parseOptions(arguments []string, definitions []optionSpec) (parsedOptions, error) {
	result := parsedOptions{Values: map[string][]string{}, Positional: []string{}}
	known := map[string]optionSpec{}
	for _, definition := range definitions {
		known[definition.Name] = definition
		if definition.Default != "" {
			result.Values[definition.Name] = []string{definition.Default}
		}
	}
	for i := 0; i < len(arguments); i++ {
		item := arguments[i]
		if item == "--" {
			result.Positional = append(result.Positional, arguments[i+1:]...)
			break
		}
		if item == "--help" || item == "-h" {
			result.Help = true
			continue
		}
		if !strings.HasPrefix(item, "-") {
			result.Positional = append(result.Positional, item)
			continue
		}
		if !strings.HasPrefix(item, "--") {
			return result, fmt.Errorf("unknown option: %s", item)
		}
		pieces := strings.SplitN(strings.TrimPrefix(item, "--"), "=", 2)
		definition, ok := known[pieces[0]]
		if !ok {
			return result, fmt.Errorf("unknown option: --%s", pieces[0])
		}
		value := ""
		if len(pieces) == 2 {
			value = pieces[1]
		} else if definition.Boolean {
			value = "true"
		} else {
			if i+1 >= len(arguments) || strings.HasPrefix(arguments[i+1], "--") {
				return result, fmt.Errorf("--%s requires a value", definition.Name)
			}
			i++
			value = arguments[i]
		}
		if definition.Boolean {
			if value != "true" && value != "false" {
				return result, fmt.Errorf("--%s must be true or false", definition.Name)
			}
		}
		if definition.Repeat {
			result.Values[definition.Name] = append(result.Values[definition.Name], value)
		} else {
			result.Values[definition.Name] = []string{value}
		}
	}
	return result, nil
}
func extractGlobals(arguments []string) ([]string, parsedOptions, error) {
	remaining, selected := []string{}, []string{}
	ended := false
	names := map[string]bool{}
	for _, definition := range globalOptions {
		names[definition.Name] = true
	}
	for i := 0; i < len(arguments); i++ {
		item := arguments[i]
		if item == "--" {
			ended = true
			remaining = append(remaining, item)
			continue
		}
		name := strings.SplitN(strings.TrimPrefix(item, "--"), "=", 2)[0]
		if !ended && strings.HasPrefix(item, "--") && names[name] {
			selected = append(selected, item)
			if !strings.Contains(item, "=") {
				if i+1 >= len(arguments) || strings.HasPrefix(arguments[i+1], "--") {
					return nil, parsedOptions{}, fmt.Errorf("%s requires a value", item)
				}
				i++
				selected = append(selected, arguments[i])
			}
		} else {
			remaining = append(remaining, item)
		}
	}
	parsed, err := parseOptions(selected, globalOptions)
	return remaining, parsed, err
}
func rootHelp(writer io.Writer) {
	fmt.Fprintln(writer, "agentforge — local-first skill audit and maintenance (Go, standard library only)")
	fmt.Fprintln(writer, "\nUsage: agentforge [global options] COMMAND [options]")
	fmt.Fprintln(writer, "\nCommands:\n  search              Find useful skills from an ordinary task description\n  audit (scan)         Inventory skills, duplicates and eager rules\n  compare             Explain differences against an explicitly chosen canonical skill\n  lint [SKILL ...]     Structural scores, script flags and lexical routing cases\n  sync dedupe|cold|restore|export|promote\n  discover prepare|inspect|fetch|search\n  drift audit|distill (automate-me)")
	fmt.Fprintln(writer, "\nManagement previews are read-only; --apply performs the explicit mutation.")
	fmt.Fprintln(writer, "audit uses configured harness roots by default; lint without SKILL paths does the same.")
	for _, definition := range globalOptions {
		fmt.Fprintf(writer, "  --%-18s %s\n", definition.Name, definition.Help)
	}
	fmt.Fprintln(writer, "  --help, -h           Show help\n  --version            Show runtime/version metadata")
}
func commandHelp(writer io.Writer, route string) {
	fmt.Fprintf(writer, "Usage: agentforge %s [options]\n", route)
	if route == "audit" || route == "lint" {
		if route == "audit" {
			fmt.Fprintln(writer, "Without --root, inventory configured harness roots automatically.")
		} else {
			fmt.Fprintln(writer, "Without SKILL paths, inventory configured roots; --root selects custom roots.")
		}
		fmt.Fprintln(writer, "--project sets workspace scope; home roots are included. Use --project \"$HOME\" for home scope.")
		fmt.Fprintln(writer, "--root selects custom skill roots and skips configured eager-rule scanning.")
		fmt.Fprintln(writer, "Audit JSON fields and estimate definitions: references/report-schema.md")
	}
	if route == "lint" {
		fmt.Fprintln(writer, "Positional arguments: zero or more skill directories or SKILL.md files")
		fmt.Fprintln(writer, "Containers without SKILL.md are reported in skipped; their discovered skills are linted.")
		fmt.Fprintln(writer, "Issue meanings and actionability: references/lint-codes.md")
	}
	if route == "discover fetch" {
		fmt.Fprintln(writer, "Positional argument: owner/repo[/path] or a public GitHub/skills.sh URL")
	}
	for _, definition := range append(append([]optionSpec{}, globalOptions...), routes[route]...) {
		suffix := ""
		if definition.Default != "" {
			suffix = " (default " + definition.Default + ")"
		}
		if definition.Repeat {
			suffix += " (repeatable)"
		}
		fmt.Fprintf(writer, "  --%-20s %s%s\n", definition.Name, definition.Help, suffix)
	}
	fmt.Fprintln(writer, "  --help, -h             Show help")
	if route == "sync cold" {
		fmt.Fprintln(writer, "Example: agentforge sync cold --skill /path/one --skill /path/two")
		fmt.Fprintln(writer, "Review the full items list, then repeat with --apply. One transaction restores the whole batch.")
	}
	if route == "sync dedupe" {
		fmt.Fprintln(writer, "Example: agentforge sync dedupe --canonical /path/main --duplicate /path/copy1 --duplicate /path/copy2")
	}
}
func require(options parsedOptions, names ...string) error {
	for _, name := range names {
		if options.value(name) == "" {
			return fmt.Errorf("--%s is required", name)
		}
	}
	return nil
}
func asFloat(value string) (float64, error) {
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("invalid number: %s", value)
	}
	return n, nil
}
func (c *TriggerCase) UnmarshalJSON(data []byte) error {
	var raw struct {
		Prompt   *string         `json:"prompt"`
		Expected json.RawMessage `json:"expected"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Prompt == nil {
		return errors.New("trigger case requires a prompt string")
	}
	c.Prompt = *raw.Prompt
	c.Expected = []string{}
	if len(raw.Expected) == 0 || string(raw.Expected) == "null" {
		return nil
	}
	if err := json.Unmarshal(raw.Expected, &c.Expected); err == nil {
		return nil
	}
	var value string
	if err := json.Unmarshal(raw.Expected, &value); err != nil {
		return errors.New("expected must be a skill name or list of names")
	}
	c.Expected = []string{value}
	return nil
}

func Run(arguments []string, stdout, stderr io.Writer) int {
	fail := func(err error) int { json.NewEncoder(stderr).Encode(M{"error": err.Error()}); return 2 }
	arguments, globals, err := extractGlobals(arguments)
	if err != nil {
		return fail(err)
	}
	if len(arguments) == 0 || arguments[0] == "--help" || arguments[0] == "-h" {
		rootHelp(stdout)
		return 0
	}
	if arguments[0] == "--version" {
		json.NewEncoder(stdout).Encode(M{"version": Version, "runtime": "go", "os": runtime.GOOS, "arch": runtime.GOARCH})
		return 0
	}
	route := arguments[0]
	arguments = arguments[1:]
	switch route {
	case "scan":
		route = "audit"
	case "sync", "discover", "drift":
		if len(arguments) == 0 || arguments[0] == "--help" || arguments[0] == "-h" {
			rootHelp(stdout)
			return 0
		}
		route += " " + arguments[0]
		arguments = arguments[1:]
	}
	if route == "drift automate-me" {
		route = "drift distill"
	}
	definitions, ok := routes[route]
	if !ok {
		return fail(fmt.Errorf("unknown command: %s", route))
	}
	options, err := parseOptions(arguments, definitions)
	if err != nil {
		return fail(err)
	}
	if options.Help {
		commandHelp(stdout, route)
		return 0
	}
	if route != "lint" && route != "discover fetch" && len(options.Positional) > 0 {
		return fail(errors.New("unexpected positional arguments"))
	}
	e, err := NewEngine(globals.value("config"))
	if err != nil {
		return fail(err)
	}
	project := globals.value("project")
	if project == "" {
		project, err = os.Getwd()
		if err != nil {
			return fail(err)
		}
	}
	project, err = realPath(project)
	if err != nil {
		return fail(err)
	}
	state := globals.value("state-dir")
	if state == "" {
		state = e.defaultState()
	}
	if strings.HasPrefix(route, "sync ") || route == "discover fetch" || route == "discover prepare" || route == "drift distill" {
		state, err = e.outsideRoots(state, project)
		if err != nil {
			return fail(err)
		}
	}
	var result M
	code := 0
	switch route {
	case "search":
		if err = require(options, "query"); err != nil {
			return fail(err)
		}
		limit, parseErr := strconv.Atoi(options.value("limit"))
		if parseErr != nil {
			return fail(errors.New("limit must be an integer"))
		}
		var constraints *SearchConstraints
		_, setupSpecified := options.Values["allow-setup"]
		_, credentialsSpecified := options.Values["allow-credentials"]
		if options.Values["available-tool"] != nil || options.boolean("no-tools") || setupSpecified || credentialsSpecified {
			constraints = &SearchConstraints{AvailableTools: options.Values["available-tool"], AllowSetup: true, AllowCredentials: true}
			if options.boolean("no-tools") {
				if len(constraints.AvailableTools) > 0 {
					return fail(errors.New("--no-tools cannot be combined with --available-tool"))
				}
				constraints.AvailableTools = []string{}
			}
			if setupSpecified {
				constraints.AllowSetup = options.boolean("allow-setup")
			}
			if credentialsSpecified {
				constraints.AllowCredentials = options.boolean("allow-credentials")
			}
		}
		result, err = CloudSearchWithConstraints(options.value("query"), options.value("harness"), options.value("endpoint"), limit, constraints)
	case "audit":
		result, err = e.Scan(project, options.Values["root"])
		if err == nil {
			if len(result["errors"].([]M)) > 0 {
				code = 1
			}
			if options.boolean("fail-on-budget") {
				for _, summary := range result["summaries"].(map[string]M) {
					if summary["over_index_budget"] == true || summary["over_eager_budget"] == true {
						code = 1
					}
				}
			}
		}
	case "compare":
		if err = require(options, "canonical"); err != nil {
			return fail(err)
		}
		result, err = e.CompareGroup(options.value("canonical"), options.Values["copy"], options.Values["root"], project,
			CompareOptions{NormalizeHarnessPaths: options.boolean("normalize-harness-paths"), ShowLines: options.boolean("show-lines")})
		if err == nil && len(result["errors"].([]M)) > 0 {
			code = 1
		}
	case "lint":
		minimum, parseErr := asFloat(options.value("fail-under"))
		if parseErr != nil {
			return fail(parseErr)
		}
		if minimum < 0 || minimum > 100 {
			return fail(errors.New("score floor must be 0..100"))
		}
		threshold := e.Config.CollisionThreshold
		if options.value("collision-threshold") != "" {
			threshold, err = asFloat(options.value("collision-threshold"))
			if err != nil {
				return fail(err)
			}
		}
		if threshold < 0 || threshold > 1 {
			return fail(errors.New("collision threshold must be 0..1"))
		}
		paths := options.Positional
		problems := []M{}
		if len(paths) == 0 {
			inventory, scanErr := e.Scan(project, options.Values["root"])
			if scanErr != nil {
				return fail(scanErr)
			}
			problems = append(problems, inventory["errors"].([]M)...)
			for _, skill := range inventory["skills"].([]M) {
				paths = append(paths, filepath.Dir(str(skill["path"])))
			}
		}
		resolved := []string{}
		skipped := []M{}
		managementByPath := map[string]string{}
		for _, path := range paths {
			if filepath.Base(path) == "SKILL.md" {
				path = filepath.Dir(path)
			}
			management := managedReason(path)
			path, err = realPath(path)
			if err != nil {
				problems = append(problems, M{"path": path, "error": err.Error()})
				continue
			}
			if management != "" {
				managementByPath[path] = management
			}
			if isDir(path) && !exists(filepath.Join(path, "SKILL.md")) {
				children, errors := walkSkills(path, e.Config.Limits["scan_dirs"])
				skipped = append(skipped, M{"path": path, "reason": "container without SKILL.md", "discovered_skills": len(children)})
				for _, problem := range errors {
					problems = append(problems, M{"path": path, "error": problem})
				}
				for _, child := range children {
					root, err := realPath(filepath.Dir(child))
					if err != nil {
						problems = append(problems, M{"path": child, "error": err.Error()})
					} else {
						resolved = append(resolved, root)
						if reason := managedReason(child); reason != "" {
							managementByPath[root] = reason
						}
					}
				}
				continue
			}
			resolved = append(resolved, path)
		}
		skills := []M{}
		for _, path := range unique(resolved) {
			skill, lintErr := e.LintSkill(path)
			if lintErr != nil {
				problems = append(problems, M{"path": path, "error": lintErr.Error()})
				continue
			}
			if management := managementByPath[path]; management != "" {
				skill["managed"], skill["managed_reason"], skill["cleanup_eligible"] = true, management, false
				skill["prune_candidates"] = []M{}
				annotateIssues(skill["issues"].([]M), true)
			}
			skills = append(skills, skill)
			if skill["sdqs"].(float64) < minimum {
				code = 1
			}
			for _, issue := range skill["issues"].([]M) {
				if issue["severity"] == "error" {
					code = 1
				}
			}
		}
		cases := []TriggerCase{}
		if path := options.value("trigger-tests"); path != "" {
			if err = readJSON(path, &cases); err != nil {
				return fail(err)
			}
			if cases == nil {
				return fail(errors.New("trigger tests must be a JSON list"))
			}
		}
		routing, routeErr := routingReport(skills, threshold, e.Config.RoutingTests.MinCosine, e.Config.RoutingTests.MinSharedTerms, cases, options.boolean("full-matrix"))
		if routeErr != nil {
			return fail(routeErr)
		}
		for _, test := range routing["trigger_tests"].([]M) {
			if test["passed"] != true {
				code = 1
			}
		}
		if len(problems) > 0 {
			code = 1
		}
		result = M{"skills": skills, "routing": routing, "errors": problems, "skipped": skipped}
	case "sync dedupe":
		if err = require(options, "canonical", "duplicate"); err != nil {
			return fail(err)
		}
		result, err = e.Archive(options.Values["duplicate"], state, options.boolean("apply"), options.value("canonical"))
	case "sync cold":
		if err = require(options, "skill"); err != nil {
			return fail(err)
		}
		result, err = e.Archive(options.Values["skill"], state, options.boolean("apply"), "")
	case "sync restore":
		if err = require(options, "transaction"); err != nil {
			return fail(err)
		}
		result, err = e.Restore(options.value("transaction"), state, options.boolean("apply"))
	case "sync export":
		if err = require(options, "skill", "harness", "destination"); err != nil {
			return fail(err)
		}
		result, err = e.ExportSkill(options.value("skill"), options.value("harness"), options.value("destination"), ExportOptions{Apply: options.boolean("apply"), Isolate: options.boolean("isolate"), AllowLossy: options.boolean("allow-lossy"), Format: options.value("format")})
	case "sync promote":
		if err = require(options, "quarantine", "destination"); err != nil {
			return fail(err)
		}
		result, err = e.Promote(options.value("quarantine"), options.value("destination"), options.value("review-sha256"), options.boolean("accept-findings"), options.boolean("apply"))
	case "discover inspect":
		if err = require(options, "path"); err != nil {
			return fail(err)
		}
		result, err = e.Inspect(options.value("path"))
	case "discover prepare":
		if err = require(options, "result", "destination", "harness"); err != nil {
			return fail(err)
		}
		index, parseErr := strconv.Atoi(options.value("index"))
		if parseErr != nil {
			return fail(errors.New("index must be an integer between 1 and 20"))
		}
		data, readErr := readPackageFile(options.value("result"), 1<<20)
		if readErr != nil {
			return fail(readErr)
		}
		result, err = e.PrepareAdoption(data, index, options.value("destination"), options.value("harness"), state, project, options.Values["root"])
	case "discover fetch":
		if len(options.Positional) != 1 {
			return fail(errors.New("fetch requires exactly one source"))
		}
		var selected *string
		if values, ok := options.Values["skill-path"]; ok {
			selected = &values[0]
		}
		result, err = e.Fetch(options.Positional[0], state, options.value("ref"), selected, options.value("archive-sha256"))
	case "discover search":
		if err = require(options, "query"); err != nil {
			return fail(err)
		}
		result, err = e.Search(options.value("query"))
	case "drift audit":
		if err = require(options, "skill", "repo"); err != nil {
			return fail(err)
		}
		result, err = e.DriftAudit(options.value("skill"), options.value("repo"), options.value("base"), options.Values["transcript"], options.Values["pr"])
	case "drift distill":
		if err = require(options, "transcript", "name", "description"); err != nil {
			return fail(err)
		}
		minimum, parseErr := strconv.Atoi(options.value("min-occurrences"))
		if parseErr != nil {
			return fail(parseErr)
		}
		result, err = e.Distill(options.Values["transcript"], options.value("name"), options.value("description"), filepath.Join(state, "drafts", options.value("name")), minimum, options.boolean("apply"))
	default:
		return fail(errors.New("unimplemented command"))
	}
	if err != nil {
		return fail(err)
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err = encoder.Encode(result); err != nil {
		return fail(err)
	}
	return code
}

// Make option presentation deterministic when consumers inspect the route catalog.
func commandNames() []string {
	names := make([]string, 0, len(routes))
	for name := range routes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
