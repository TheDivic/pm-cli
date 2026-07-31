// Package ignore applies .gitignore semantics to discovery without requiring
// the git executable. It reads every .gitignore from the root downward and
// matches paths with nested, negated, and directory-pattern support via the
// pure-Go go-git matcher.
package ignore

import (
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// Matcher tests paths against the .gitignore rules found under a root.
type Matcher struct {
	m gitignore.Matcher
}

// New reads all .gitignore patterns under root and returns a matcher. A nil
// Matcher (or one with no patterns) ignores nothing.
func New(root string) (*Matcher, error) {
	patterns, err := gitignore.ReadPatterns(osfs.New(root), nil)
	if err != nil {
		return nil, err
	}
	return &Matcher{m: gitignore.NewMatcher(patterns)}, nil
}

// Ignored reports whether relPath (relative to the root, OS-separated) is
// ignored. isDir must say whether the path is a directory.
func (m *Matcher) Ignored(relPath string, isDir bool) bool {
	if m == nil || relPath == "" || relPath == "." {
		return false
	}
	return m.m.Match(components(relPath), isDir)
}

func components(relPath string) []string {
	return strings.Split(strings.ReplaceAll(relPath, "\\", "/"), "/")
}
