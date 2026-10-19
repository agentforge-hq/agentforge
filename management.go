package agentforge

import (
	"path/filepath"
	"strings"
)

func managedReason(path string) string {
	paths := []string{path}
	if resolved, err := realPath(path); err == nil {
		paths = append(paths, resolved)
	}
	for _, candidate := range paths {
		clean := "/" + strings.ToLower(filepath.ToSlash(filepath.Clean(candidate))) + "/"
		switch {
		case strings.Contains(clean, "/plugins/cache/"):
			return "plugin-cache: maintained by the harness or plugin installer"
		case strings.Contains(clean, "/skills/.system/"):
			return "built-in: maintained by the harness"
		case strings.Contains(clean, "/.claude/skills/synced/"):
			return "synced: maintained by Claude synchronization"
		}
	}
	return ""
}
