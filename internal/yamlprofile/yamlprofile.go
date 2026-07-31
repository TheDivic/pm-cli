// Package yamlprofile parses task-file bytes into a YAML node tree and enforces
// the restricted YAML 1.2 profile the format requires: one document, no
// anchors, aliases, merge keys, explicit or unsupported tags, and no duplicate
// mapping keys. Syntax-level checks run here before any typed decoding.
package yamlprofile

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// allowedTags are the resolved YAML tags the profile permits. Anything else
// (for example !!bool, !!float, !!timestamp from an unquoted date, !!binary, or
// a custom tag) is rejected.
var allowedTags = map[string]bool{
	"!!str":  true,
	"!!int":  true,
	"!!null": true,
	"!!map":  true,
	"!!seq":  true,
}

// Load parses data as a single YAML document, enforces the restricted profile,
// and returns the top-level mapping node ready for decoding.
func Load(data []byte) (*yaml.Node, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))

	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("a task file must contain exactly one YAML document")
		}
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	var extra yaml.Node
	switch err := dec.Decode(&extra); {
	case err == nil:
		return nil, fmt.Errorf("line %d: multiple YAML documents are not allowed", extra.Line)
	case !errors.Is(err, io.EOF):
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 {
		return nil, errors.New("expected a single YAML document with one root value")
	}
	root := doc.Content[0]
	if err := check(root); err != nil {
		return nil, err
	}
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("line %d: the document root must be a mapping", root.Line)
	}
	return root, nil
}

// check walks the node tree rejecting every construct outside the profile.
func check(n *yaml.Node) error {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.AliasNode {
		return fmt.Errorf("line %d: YAML aliases are not allowed", n.Line)
	}
	if n.Anchor != "" {
		return fmt.Errorf("line %d: YAML anchors are not allowed", n.Line)
	}
	if n.Tag != "" && !allowedTags[n.Tag] {
		return fmt.Errorf("line %d: unsupported or explicit YAML tag %q", n.Line, n.Tag)
	}

	switch n.Kind {
	case yaml.MappingNode:
		seen := make(map[string]bool, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, val := n.Content[i], n.Content[i+1]
			if key.Tag == "!!merge" || key.Value == "<<" {
				return fmt.Errorf("line %d: YAML merge keys are not allowed", key.Line)
			}
			if key.Kind == yaml.ScalarNode {
				if seen[key.Value] {
					return fmt.Errorf("line %d: duplicate mapping key %q", key.Line, key.Value)
				}
				seen[key.Value] = true
			}
			if err := check(key); err != nil {
				return err
			}
			if err := check(val); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for _, c := range n.Content {
			if err := check(c); err != nil {
				return err
			}
		}
	}
	return nil
}
