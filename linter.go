package agentforge

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var stopwords = strings.Fields("a an and are as at be by for from in is it of on or that the this to use when with do not")
var wordRE = regexp.MustCompile(`[a-z0-9]+`)
var localPathRE = regexp.MustCompile(`(?:scripts|references|assets)/[\w./-]+\.[A-Za-z0-9]+`)
var portableFields = []string{"name", "description", "license", "compatibility", "metadata", "allowed-tools"}

func words(text string) []string {
	result := []string{}
	for _, word := range wordRE.FindAllString(strings.ToLower(text), -1) {
		if len(word) > 1 && !contains(stopwords, word) {
			result = append(result, word)
		}
	}
	return result
}
func frequencies(values []string) map[string]int {
	result := map[string]int{}
	for _, v := range values {
		result[v]++
	}
	return result
}
func localReferences(text string) []string { return unique(localPathRE.FindAllString(text, -1)) }

type docLine struct {
	Number         int
	Text, Language string
	Fenced         bool
}

func documentLines(text string) []docLine {
	result := []docLine{}
	fence, language := "", ""
	marker := regexp.MustCompile("^[ \\t]{0,3}(\x60{3,}|~{3,})(.*)$")
	for i, line := range strings.Split(text, "\n") {
		match := marker.FindStringSubmatch(line)
		if match != nil {
			delimiter, info := match[1], strings.TrimSpace(match[2])
			if fence == "" {
				fence = delimiter
				language = ""
				if fields := strings.Fields(info); len(fields) > 0 {
					language = strings.ToLower(fields[0])
				}
				continue
			}
			if delimiter[0] == fence[0] && len(delimiter) >= len(fence) && info == "" {
				fence, language = "", ""
				continue
			}
		}
		result = append(result, docLine{i + 1, line, language, fence != ""})
	}
	return result
}
func shellWords(line string) ([]string, error) {
	result := []string{}
	var current strings.Builder
	quote := rune(0)
	escaped, started := false, false
	for _, r := range line {
		if escaped {
			current.WriteRune(r)
			escaped = false
			started = true
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
			started = true
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			started = true
			continue
		}
		if unicode.IsSpace(r) {
			if started {
				result = append(result, current.String())
				current.Reset()
				started = false
			}
			continue
		}
		current.WriteRune(r)
		started = true
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("unclosed shell quotation")
	}
	if started {
		result = append(result, current.String())
	}
	return result, nil
}

type scriptCommand struct {
	Script  string
	Options []string
	Line    int
}

func scriptCommands(text string) []scriptCommand {
	result := []scriptCommand{}
	scriptRE := regexp.MustCompile(`^(?:\./)?scripts/[\w./-]+$`)
	runtimeRE := regexp.MustCompile(`(?i)^(?:python(?:\d+(?:\.\d+)*)?|node|bash|sh|run|execute|invoke)$`)
	for _, line := range documentLines(strings.ReplaceAll(text, "\\\n", " ")) {
		if !strings.Contains(line.Text, "scripts/") || (line.Fenced && !contains([]string{"", "sh", "bash", "shell", "console"}, line.Language)) {
			continue
		}
		if regexp.MustCompile(`(?i)^\s*(?:[-*+]\s+)?examples?\s*:`).MatchString(line.Text) {
			continue
		}
		items, err := shellWords(strings.Trim(strings.TrimSpace(line.Text), "\x60"))
		if err != nil {
			continue
		}
		for i, item := range items {
			item = strings.Trim(item, "\x60")
			if !scriptRE.MatchString(item) {
				continue
			}
			invocation := i == 0
			for j := 0; j < i; j++ {
				if runtimeRE.MatchString(strings.Trim(items[j], "\x60")) {
					onlyFlags := true
					for _, option := range items[j+1 : i] {
						if !strings.HasPrefix(option, "-") {
							onlyFlags = false
						}
					}
					if onlyFlags {
						invocation = true
					}
				}
			}
			if !invocation {
				continue
			}
			options := []string{}
			for _, option := range items[i+1:] {
				if contains([]string{";", "&&", "||", "|", ">", ">>", "#"}, option) {
					break
				}
				if regexp.MustCompile(`^--?[A-Za-z]`).MatchString(option) {
					options = append(options, strings.TrimRight(strings.SplitN(option, "=", 2)[0], "\x60.,"))
				}
			}
			result = append(result, scriptCommand{strings.TrimPrefix(item, "./"), options, line.Number})
		}
	}
	return result
}
func requiredReferences(text string) []string {
	result := []string{}
	for _, command := range scriptCommands(text) {
		result = append(result, command.Script)
	}
	linkRE := regexp.MustCompile(`\[[^\]]*\]\(\s*<?(?:\./)?((?:scripts|references|assets)/[\w./-]+\.[A-Za-z0-9]+)(?:#[^\s)>]*)?>?\s*\)`)
	instructionRE := regexp.MustCompile("(?i)^\\s*(?:(?:[-*+]|\\d+[.)])\\s+)?(?:read|consult|see|follow|load|review)\\s+(?:(?:the|this|a|an)\\s+)?\x60(?:\\./)?((?:scripts|references|assets)/[\\w./-]+\\.[A-Za-z0-9]+)\x60")
	for _, line := range documentLines(text) {
		if line.Fenced {
			continue
		}
		for _, match := range linkRE.FindAllStringSubmatch(line.Text, -1) {
			result = append(result, match[1])
		}
		if match := instructionRE.FindStringSubmatch(line.Text); match != nil {
			result = append(result, match[1])
		}
	}
	return unique(result)
}

type lexeme struct {
	Kind, Value string
	Line        int
}

// Python compatibility inspection is deliberately lexical, never advertised as a Python AST.
func pythonLex(text string) ([]lexeme, error) {
	result := []lexeme{}
	line := 1
	stack := []byte{}
	for i := 0; i < len(text); {
		c := text[i]
		if c == '\n' {
			line++
			i++
			continue
		}
		if c == '#' {
			for i < len(text) && text[i] != '\n' {
				i++
			}
			continue
		}
		if c == '\'' || c == '"' {
			startLine := line
			quote := c
			triple := i+2 < len(text) && text[i+1] == quote && text[i+2] == quote
			n := 1
			if triple {
				n = 3
			}
			i += n
			var value strings.Builder
			closed := false
			for i < len(text) {
				if text[i] == '\\' && i+1 < len(text) {
					if text[i+1] == '\n' {
						line++
					}
					value.WriteByte(text[i+1])
					i += 2
					continue
				}
				if text[i] == quote && (!triple || (i+2 < len(text) && text[i+1] == quote && text[i+2] == quote)) {
					i += n
					closed = true
					break
				}
				if text[i] == '\n' {
					if !triple {
						return nil, fmt.Errorf("unterminated Python string at line %d", startLine)
					}
					line++
				}
				value.WriteByte(text[i])
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated Python string at line %d", startLine)
			}
			result = append(result, lexeme{"string", value.String(), startLine})
			continue
		}
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
			start := i
			i++
			for i < len(text) && ((text[i] >= 'a' && text[i] <= 'z') || (text[i] >= 'A' && text[i] <= 'Z') || (text[i] >= '0' && text[i] <= '9') || text[i] == '_') {
				i++
			}
			result = append(result, lexeme{"identifier", text[start:i], line})
			continue
		}
		if strings.ContainsRune("([{", rune(c)) {
			stack = append(stack, c)
		}
		if strings.ContainsRune(")]}", rune(c)) {
			if len(stack) == 0 || strings.IndexByte("([{", stack[len(stack)-1]) != strings.IndexByte(")]}", c) {
				return nil, fmt.Errorf("unbalanced Python delimiter at line %d", line)
			}
			stack = stack[:len(stack)-1]
		}
		if !unicode.IsSpace(rune(c)) {
			result = append(result, lexeme{"punctuation", string(c), line})
		}
		i++
	}
	if len(stack) > 0 {
		return nil, fmt.Errorf("unclosed Python delimiter")
	}
	return result, nil
}

func inspectPython(text string) (M, error) {
	items, err := pythonLex(text)
	if err != nil {
		return nil, err
	}
	flags := map[string]bool{"-h": true, "--help": true}
	declarations := []M{}
	dynamic := false
	for i, item := range items {
		if item.Kind != "identifier" || item.Value != "add_argument" || i+1 >= len(items) || items[i+1].Value != "(" {
			continue
		}
		depth := 1
		options := []string{}
		argument := []lexeme{}
		consume := func() {
			if len(argument) == 0 {
				return
			}
			if len(argument) == 1 && argument[0].Kind == "string" {
				if strings.HasPrefix(argument[0].Value, "-") {
					flags[argument[0].Value] = true
					options = append(options, argument[0].Value)
				}
			} else if len(argument) < 2 || argument[1].Value != "=" {
				dynamic = true
			}
			argument = nil
		}
		for j := i + 2; j < len(items); j++ {
			current := items[j]
			if current.Kind == "punctuation" && strings.Contains("([{", current.Value) {
				depth++
			}
			if current.Kind == "punctuation" && strings.Contains(")]}", current.Value) {
				depth--
				if depth == 0 {
					consume()
					break
				}
			}
			if depth == 1 && current.Value == "," {
				consume()
			} else {
				argument = append(argument, current)
			}
		}
		declarations = append(declarations, M{"line": item.Line, "options": options})
	}
	return M{"flags": sortedKeys(flags), "dynamic_options": dynamic, "declarations": declarations, "verification": "Python literal add_argument inspection; not full syntax or AST validation"}, nil
}
func inspectGo(path, text string) (M, error) {
	set := token.NewFileSet()
	tree, err := parser.ParseFile(set, path, text, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	flags := map[string]bool{"-h": true, "--help": true}
	declarations := []M{}
	dynamic := false
	methods := []string{"String", "Bool", "Int", "Int64", "Uint", "Uint64", "Float64", "Duration", "StringVar", "BoolVar", "IntVar", "Int64Var", "UintVar", "Uint64Var", "Float64Var", "DurationVar", "Var", "Func", "BoolFunc"}
	ast.Inspect(tree, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !contains(methods, selector.Sel.Name) {
			return true
		}
		index := 0
		if strings.HasSuffix(selector.Sel.Name, "Var") || selector.Sel.Name == "Var" {
			index = 1
		}
		if len(call.Args) <= index {
			dynamic = true
			return true
		}
		literal, ok := call.Args[index].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			dynamic = true
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			dynamic = true
			return true
		}
		option := value
		if !strings.HasPrefix(option, "-") {
			option = "--" + option
			flags["-"+value] = true
		}
		flags[option] = true
		declarations = append(declarations, M{"line": set.Position(call.Pos()).Line, "options": []string{option}})
		return true
	})
	return M{"flags": sortedKeys(flags), "dynamic_options": dynamic, "declarations": declarations, "verification": "Go AST syntax and literal flag declarations; no inspected code executed"}, nil
}

func (e *Engine) LintSkill(root string) (M, error) {
	management := managedReason(root)
	root, err := realPath(root)
	if err != nil {
		return nil, err
	}
	text, err := readText(filepath.Join(root, "SKILL.md"), e.Config.Limits["file_bytes"])
	if err != nil {
		return nil, err
	}
	fm := parseFrontmatter(text)
	issues, prune := []M{}, []M{}
	issue := func(code, message, severity string) {
		issues = append(issues, M{"code": code, "message": message, "severity": severity})
	}
	for _, problem := range fm.Problems {
		issue("frontmatter", problem, "error")
	}
	name, description := str(fm.Fields["name"]), str(fm.Fields["description"])
	if !validName(name) {
		issue("name", "Name must be lowercase kebab-case, 1-64 characters", "error")
	} else if name != filepath.Base(root) {
		issue("name-directory", "Name must match its directory for portable installation", "error")
	}
	if len([]rune(description)) < 1 || len([]rune(description)) > 1024 {
		issue("description", "Description must be a string of 1-1024 characters", "error")
		description = ""
	}
	if value, ok := fm.Fields["compatibility"]; ok {
		if n := len([]rune(str(value))); n < 1 || n > 500 {
			issue("compatibility", "Compatibility must be a string of 1-500 characters", "error")
		}
	}
	extensions := []string{}
	for _, field := range fm.Order {
		if !contains(portableFields, field) {
			extensions = append(extensions, field)
		}
	}
	if len(extensions) > 0 {
		sort.Strings(extensions)
		issue("harness-extensions", "Harness-specific fields require an export check: "+strings.Join(extensions, ", "), "info")
	}
	if !regexp.MustCompile(`(?i)\b(when|for|use|if)\b`).MatchString(description) {
		issue("positive-trigger", "Description lacks a clear positive routing trigger", "warning")
	}
	if !regexp.MustCompile(`(?i)\b(not for|do not|don.t use|exclude|except|instead)\b`).MatchString(description) {
		issue("routing-boundary", "Consider a negative trigger if a neighboring skill overlaps", "info")
	}
	if len(words(description)) < 5 {
		issue("vague-description", "Description has too few specific terms for reliable routing", "warning")
	}
	if tokens(description) > e.Config.Budgets["description_tokens"] {
		issue("description-budget", "Description exceeds the configured index token proxy budget", "warning")
	}
	if tokens(fm.Body) > e.Config.Budgets["skill_tokens"] {
		issue("body-budget", "Move conditional detail into linked references", "warning")
	}
	if name == "agentforge" && tokens(text) >= e.Config.Budgets["router_tokens"] {
		issue("router-budget", "agentforge root must stay below its router budget", "error")
	}
	normalized := map[string]int{}
	nonword := regexp.MustCompile(`[^\pL\pN_]+`)
	for i, line := range strings.Split(fm.Body, "\n") {
		key := strings.TrimSpace(nonword.ReplaceAllString(strings.ToLower(line), " "))
		if len([]rune(key)) < 45 || strings.HasPrefix(strings.TrimSpace(line), "#") || strings.HasPrefix(strings.TrimSpace(line), "\x60\x60\x60") || strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		if prior, ok := normalized[key]; ok {
			prune = append(prune, M{"line_in_body": i + 1, "duplicate_of": prior, "text": strings.TrimSpace(line)})
		} else {
			normalized[key] = i + 1
		}
	}
	if len(prune) > 0 {
		issue("repeated-instructions", fmt.Sprintf("%d duplicate instruction lines; review deletion candidates", len(prune)), "warning")
	}
	documents := map[string]string{"SKILL.md": text}
	dependencies := requiredReferences(text)
	unverified := []string{}
	for _, reference := range localReferences(text) {
		if !contains(dependencies, reference) {
			unverified = append(unverified, reference)
		}
	}
	for _, relative := range dependencies {
		target, err := realPath(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return nil, err
		}
		if runtime.GOOS == "windows" && filepath.Ext(relative) == "" && !isFile(target) && isFile(target+".exe") {
			target += ".exe"
		}
		if !within(target, root) {
			issue("escaping-reference", "Reference escapes skill: "+relative, "error")
		} else if !isFile(target) {
			issue("missing-reference", "Missing file: "+relative, "error")
		} else if filepath.Ext(target) == ".md" {
			content, err := readText(target, e.Config.Limits["file_bytes"])
			if err != nil {
				return nil, err
			}
			documents[relative] = content
		}
	}
	scripts := map[string]M{}
	pythonSeen := false
	visited := 0
	scriptRoot := filepath.Join(root, "scripts")
	if isDir(scriptRoot) {
		err = filepath.WalkDir(scriptRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			visited++
			if visited > e.Config.Limits["tree_files"] {
				return fmt.Errorf("script inventory limit reached")
			}
			if entry.IsDir() {
				if ignored[entry.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			extension := filepath.Ext(path)
			if extension != ".py" && extension != ".go" {
				return nil
			}
			resolved, err := realPath(path)
			if err != nil {
				return err
			}
			if !within(resolved, root) {
				issue("escaping-script", "Script resolves outside skill: "+path, "error")
				return nil
			}
			relative, _ := filepath.Rel(root, path)
			relative = filepath.ToSlash(relative)
			content, err := readText(path, e.Config.Limits["file_bytes"])
			if err != nil {
				return err
			}
			var record M
			if extension == ".py" {
				pythonSeen = true
				record, err = inspectPython(content)
			} else {
				record, err = inspectGo(path, content)
			}
			if err != nil {
				code := "go-syntax"
				if extension == ".py" {
					code = "python-lexical-error"
				}
				issue(code, relative+": "+err.Error(), "error")
			} else {
				scripts[relative] = record
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for _, relative := range sortedKeys(documents) {
		for _, command := range scriptCommands(documents[relative]) {
			if filepath.Ext(command.Script) != ".py" && filepath.Ext(command.Script) != ".go" {
				continue
			}
			record, ok := scripts[command.Script]
			if !ok {
				issues = append(issues, M{"code": "unverified-command", "message": "No inspectable script: " + command.Script, "severity": "error", "file": relative, "line": command.Line})
				continue
			}
			for _, option := range command.Options {
				if !contains(stringsOf(record["flags"]), option) {
					severity := "error"
					message := command.Script + " does not declare " + option
					if record["dynamic_options"] == true {
						severity = "warning"
						message += "; dynamic declarations need manual verification"
					}
					issues = append(issues, M{"code": "script-flag", "message": message, "severity": severity, "file": relative, "line": command.Line})
				}
			}
		}
	}
	if pythonSeen {
		issue("python-verification-boundary", "Python checks cover literals and balanced tokens; full Python syntax is unverified", "info")
	}
	if len(scripts) > 0 && !regexp.MustCompile(`--help|python\d*|scripts/`).MatchString(fm.Body) {
		issue("script-routing", "Root does not tell the agent how to use its scripts", "warning")
	}
	if !regexp.MustCompile(`(?i)\b(test|verif|validat|check|eval)`).MatchString(fm.Body) {
		issue("verification", "No observable verification step in the root playbook", "warning")
	}
	if regexp.MustCompile(`(?i)curl[^\n]*\|\s*(?:sh|bash)|ignore (?:all )?(?:previous|system) instructions`).MatchString(fm.Body) {
		issue("unsafe-instruction", "Review instruction override or remote execution pattern", "error")
	}
	annotateIssues(issues, management != "")
	penalties := map[string][]string{
		"structure":   {"frontmatter", "name", "name-directory", "description", "compatibility"},
		"routing":     {"positive-trigger", "vague-description", "description-budget"},
		"economy":     {"body-budget", "router-budget", "repeated-instructions"},
		"execution":   {"go-syntax", "python-lexical-error", "script-flag", "unverified-command", "script-routing"},
		"maintenance": {"missing-reference", "escaping-reference", "verification"},
		"safety":      {"unsafe-instruction", "escaping-script"},
	}
	dimensions := map[string]float64{}
	score := 0.0
	for dimension, codes := range penalties {
		loss := 0.0
		for _, item := range issues {
			if !contains(codes, str(item["code"])) {
				continue
			}
			switch str(item["severity"]) {
			case "error":
				loss += 0.5
			case "warning":
				loss += 0.25
			}
		}
		value := math.Round(e.Config.Weights[dimension]*math.Max(0, 1-loss)*100) / 100
		dimensions[dimension] = value
		score += value
	}
	if management != "" {
		prune = []M{}
	}
	return M{"path": root, "name": name, "description": description, "sdqs": math.Round(score*100) / 100, "dimensions": dimensions, "body_tokens_estimate": tokens(fm.Body), "issues": issues, "prune_candidates": prune, "unverified_references": unverified, "scripts": scripts,
		"managed": management != "", "managed_reason": management, "cleanup_eligible": management == "",
		"limits": "Heuristic SDQS and lexical routing. Go AST and Python literal flags are pooled across subcommands; inspected scripts are never executed. Full Python syntax is unverified."}, nil
}

type TriggerCase struct {
	Prompt   string   `json:"prompt"`
	Expected []string `json:"expected"`
}

func normalize(vector map[string]float64) {
	norm := 0.0
	for _, v := range vector {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return
	}
	for k, v := range vector {
		vector[k] = v / norm
	}
}
func cosine(left, right map[string]float64) float64 {
	score := 0.0
	for _, key := range sortedKeys(left) {
		score += left[key] * right[key]
	}
	return score
}
func routingReport(skills []M, threshold, minCosine float64, minShared int, cases []TriggerCase, full bool) (M, error) {
	if threshold < 0 || threshold > 1 || minCosine < 0 || minCosine > 1 || math.IsNaN(minCosine) || math.IsNaN(threshold) || minShared < 1 {
		return nil, fmt.Errorf("invalid collision or routing abstention thresholds")
	}
	counts := []map[string]int{}
	frequency := map[string]int{}
	for _, skill := range skills {
		count := frequencies(words(str(skill["description"])))
		counts = append(counts, count)
		for word := range count {
			frequency[word]++
		}
	}
	idf := map[string]float64{}
	for word, n := range frequency {
		idf[word] = math.Log(float64(1+len(skills))/float64(1+n)) + 1
	}
	vectors := []map[string]float64{}
	for _, count := range counts {
		vector := map[string]float64{}
		for word, n := range count {
			vector[word] = (1 + math.Log(float64(n))) * idf[word]
		}
		normalize(vector)
		vectors = append(vectors, vector)
	}
	collisions := []M{}
	for i := range skills {
		for j := i + 1; j < len(skills); j++ {
			score := cosine(vectors[i], vectors[j])
			if score >= threshold {
				collisions = append(collisions, M{"left": skills[i]["name"], "right": skills[j]["name"], "left_path": skills[i]["path"], "right_path": skills[j]["path"], "cosine": math.Round(score*10000) / 10000})
			}
		}
	}
	sort.SliceStable(collisions, func(i, j int) bool { return collisions[i]["cosine"].(float64) > collisions[j]["cosine"].(float64) })
	tests := []M{}
	for _, case_ := range cases {
		query := map[string]float64{}
		for word, n := range frequencies(words(case_.Prompt)) {
			weight, ok := idf[word]
			if !ok {
				weight = math.Log(float64(1+len(skills))) + 1
			}
			query[word] = (1 + math.Log(float64(n))) * weight
		}
		normalize(query)
		type match struct {
			Name   string
			Score  float64
			Shared int
		}
		ranked := []match{}
		for i, skill := range skills {
			shared := 0
			for word := range query {
				if _, ok := vectors[i][word]; ok {
					shared++
				}
			}
			ranked = append(ranked, match{str(skill["name"]), cosine(vectors[i], query), shared})
		}
		sort.SliceStable(ranked, func(i, j int) bool {
			if ranked[i].Score == ranked[j].Score {
				return ranked[i].Name < ranked[j].Name
			}
			return ranked[i].Score > ranked[j].Score
		})
		winners := []string{}
		best := -1.0
		for _, item := range ranked {
			if item.Score <= 0 || item.Score < minCosine || item.Shared < minShared {
				continue
			}
			if best < 0 {
				best = item.Score
			}
			if math.Abs(item.Score-best) < 1e-10 {
				winners = append(winners, item.Name)
			}
		}
		winners = unique(winners)
		passed := len(winners) == 0
		if len(case_.Expected) > 0 {
			passed = len(winners) == 1 && contains(case_.Expected, winners[0])
		}
		ranking := []M{}
		for i, item := range ranked {
			if i >= 3 {
				break
			}
			ranking = append(ranking, M{"name": item.Name, "score": math.Round(item.Score*10000) / 10000, "shared_terms": item.Shared})
		}
		expected := case_.Expected
		if expected == nil {
			expected = []string{}
		}
		tests = append(tests, M{"prompt": case_.Prompt, "expected": expected, "winners": winners, "passed": passed, "ranking": ranking})
	}
	result := M{"method": "TF-IDF cosine; lexical proxy, not an LLM trigger evaluation", "threshold": threshold, "abstention": M{"min_cosine": minCosine, "min_shared_terms": minShared}, "collisions": collisions, "trigger_tests": tests}
	if full {
		paths := []string{}
		matrix := [][]float64{}
		for i, skill := range skills {
			paths = append(paths, str(skill["path"]))
			row := []float64{}
			for j := range skills {
				row = append(row, math.Round(cosine(vectors[i], vectors[j])*10000)/10000)
			}
			matrix = append(matrix, row)
		}
		result["matrix"] = M{"paths": paths, "cosine": matrix}
	}
	return result, nil
}
