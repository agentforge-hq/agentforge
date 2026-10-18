package agentforge

import (
	"encoding/json"
	"regexp"
	"strings"
)

type Frontmatter struct {
	Fields   M
	Body     string
	Blocks   map[string]string
	Order    []string
	Problems []string
}

var keyRE = regexp.MustCompile(`^([A-Za-z][\w-]*)\s*:`)
var blockRE = regexp.MustCompile(`^[>|][+-]?(\s+#.*)?$`)
var singleRE = regexp.MustCompile(`^'((?:[^']|'')*)'\s*(?:#.*)?$`)
var commentRE = regexp.MustCompile(`\s+#`)

func parseFrontmatter(text string) Frontmatter {
	result := Frontmatter{Fields: M{}, Body: text, Blocks: map[string]string{}, Order: []string{}, Problems: []string{}}
	lines := strings.SplitAfter(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		result.Problems = append(result.Problems, "Missing YAML frontmatter")
		return result
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		result.Problems = append(result.Problems, "Unclosed YAML frontmatter")
		return result
	}
	result.Body = strings.Join(lines[end+1:], "")
	starts := []int{}
	for i := 1; i < end; i++ {
		if keyRE.MatchString(lines[i]) {
			starts = append(starts, i)
		}
	}
	for position, start := range starts {
		stop := end
		if position+1 < len(starts) {
			stop = starts[position+1]
		}
		pieces := strings.SplitN(lines[start], ":", 2)
		key, raw := strings.TrimSpace(pieces[0]), strings.TrimSpace(pieces[1])
		if _, ok := result.Fields[key]; ok {
			result.Problems = append(result.Problems, "Duplicate frontmatter key: "+key)
		} else {
			result.Order = append(result.Order, key)
		}
		result.Blocks[key] = strings.Join(lines[start:stop], "")
		continuation := lines[start+1 : stop]
		content := []string{}
		for _, line := range continuation {
			if !strings.HasPrefix(strings.TrimSpace(line), "#") {
				content = append(content, strings.TrimSpace(line))
			}
		}
		switch {
		case blockRE.MatchString(raw):
			separator := "\n"
			if strings.HasPrefix(raw, ">") {
				separator = " "
			}
			result.Fields[key] = strings.TrimSpace(strings.Join(content, separator))
		case raw == "":
			result.Fields[key] = M{"opaque_yaml": strings.Join(continuation, "")}
		case strings.HasPrefix(raw, `"`):
			var value any
			if err := json.NewDecoder(strings.NewReader(raw)).Decode(&value); err != nil {
				result.Fields[key] = ""
				result.Problems = append(result.Problems, "Cannot read "+key+": "+err.Error())
			} else {
				result.Fields[key] = value
			}
		case strings.HasPrefix(raw, "'"):
			match := singleRE.FindStringSubmatch(raw)
			if match == nil {
				result.Fields[key] = ""
				result.Problems = append(result.Problems, "Cannot read "+key+": unclosed YAML single-quoted string")
			} else {
				result.Fields[key] = strings.ReplaceAll(match[1], "''", "'")
			}
		case strings.ContainsAny(raw[:1], "[{&*!"):
			result.Fields[key] = M{"opaque_yaml": raw}
			if contains([]string{"name", "description", "compatibility"}, key) {
				result.Problems = append(result.Problems, "Unsupported YAML scalar for "+key+"; use a plain, quoted or block string")
			}
		case raw == "true" || raw == "false":
			result.Fields[key] = raw == "true"
		default:
			value := commentRE.Split(raw, 2)[0]
			if contains([]string{"name", "description", "compatibility"}, key) {
				extra := []string{}
				for _, part := range content {
					if part != "" {
						extra = append(extra, part)
					}
				}
				if len(extra) > 0 {
					value += " " + strings.Join(extra, " ")
				}
			}
			result.Fields[key] = value
		}
	}
	for _, line := range lines[1:end] {
		if strings.TrimSpace(line) != "" && line[0] != ' ' && line[0] != '\t' && line[0] != '#' && !keyRE.MatchString(line) {
			result.Problems = append(result.Problems, "Unsupported top-level YAML syntax")
		}
	}
	return result
}
