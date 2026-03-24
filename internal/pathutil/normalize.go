package pathutil

import (
	"strings"
)

// Normalize normalizes a URL path for use as a node key.
// The first path segment (domain) is lowercased; subsequent segments are kept as-is.
// Leading/trailing slashes are stripped.
func Normalize(path string) string {
	// Strip leading and trailing slashes
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}

	segments := strings.SplitN(path, "/", 2)
	// Lowercase the first segment (domain)
	segments[0] = strings.ToLower(segments[0])

	return strings.Join(segments, "/")
}

// AncestorPaths returns all ancestor paths for a given normalized path,
// ordered from nearest to farthest (parent first).
func AncestorPaths(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}

	segments := strings.Split(path, "/")
	ancestors := make([]string, 0, len(segments)-1)

	for i := len(segments) - 1; i > 0; i-- {
		ancestors = append(ancestors, strings.Join(segments[:i], "/"))
	}

	return ancestors
}
