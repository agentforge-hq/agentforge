package agentforge

type issueDefinition struct {
	Meaning, Action, Actionability string
}

var issueDefinitions = map[string]issueDefinition{
	"frontmatter":                  {"YAML frontmatter is missing, malformed or unsupported.", "Fix the reported frontmatter syntax.", "fix"},
	"name":                         {"The name is not lowercase kebab-case of 1-64 characters.", "Choose a valid portable skill name.", "fix"},
	"name-directory":               {"The declared name differs from the installation directory.", "Align the name and directory in a portable export; check references before renaming.", "fix"},
	"description":                  {"The description is absent or outside the supported length.", "Write a description of 1-1024 characters.", "fix"},
	"compatibility":                {"The compatibility scalar has an invalid length or type.", "Use a compatibility string of 1-500 characters.", "fix"},
	"harness-extensions":           {"Frontmatter contains fields outside the portable specification.", "Review field support when exporting; retain valid harness-specific behavior.", "conditional"},
	"positive-trigger":             {"The description lacks an explicit positive trigger.", "Review routing against representative tasks before changing the description.", "review"},
	"routing-boundary":             {"No explicit exclusion or disambiguation clause was found.", "Review only when a neighboring skill overlaps.", "conditional"},
	"vague-description":            {"The description contains few specific terms.", "Test whether the intended tasks select the right skill.", "review"},
	"description-budget":           {"The description exceeds its configured byte-based token budget.", "Review a shorter description with positive and negative tasks.", "review"},
	"body-budget":                  {"The root body exceeds its configured token estimate budget.", "Review moving conditional detail to references; validate preserved behavior.", "review"},
	"router-budget":                {"AgentForge's root exceeds its configured router budget.", "Shorten the router or move conditional detail into a reference.", "fix"},
	"repeated-instructions":        {"Long instruction lines repeat after text normalization.", "Inspect the reported lines before removing duplication.", "review"},
	"escaping-reference":           {"A dependency resolves outside the skill package.", "Review the dependency and package a safe local reference.", "fix"},
	"missing-reference":            {"An explicit local dependency is missing.", "Correct the path or include the required file.", "fix"},
	"escaping-script":              {"A script resolves outside the package.", "Review the link and keep inspected scripts inside the package.", "fix"},
	"go-syntax":                    {"A Go helper fails Go AST parsing.", "Fix the reported Go syntax and rerun lint.", "fix"},
	"python-lexical-error":         {"A Python helper has malformed quoted or balanced tokens.", "Fix the reported tokens; full Python syntax still needs separate validation.", "fix"},
	"unverified-command":           {"A documented command has no inspectable script.", "Include or repair the referenced script before relying on the command.", "fix"},
	"script-flag":                  {"A documented flag has no verified literal declaration.", "Correct the flag or manually verify dynamic declarations.", "review"},
	"python-verification-boundary": {"Full Python syntax was not validated.", "Use a Python-aware check when full syntax validation is required.", "informational"},
	"script-routing":               {"The root does not explain how to invoke its helpers.", "Add a concise invocation or helper reference.", "review"},
	"verification":                 {"No observable verification step was found in the root.", "Add a check tied to the skill's output.", "review"},
	"unsafe-instruction":           {"An instruction-override or remote-execution pattern needs inspection.", "Review the actual instruction; the pattern is not a trust verdict.", "review"},
}

func annotateIssues(issues []M, managed bool) {
	for _, issue := range issues {
		definition := issueDefinitions[str(issue["code"])]
		actionability := definition.Actionability
		action := definition.Action
		if managed {
			actionability = "upstream"
			action = "Report to the owning harness or plugin; do not edit this managed copy in place."
		}
		issue["meaning"] = definition.Meaning
		issue["actionability"] = actionability
		issue["actionable"] = actionability == "fix" || actionability == "review"
		issue["action"] = action
	}
}
