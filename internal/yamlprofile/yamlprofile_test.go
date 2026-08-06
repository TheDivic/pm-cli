package yamlprofile

import "testing"

const validDoc = `schema-version: 1

project:
  id: demo
  title: Demo
  task-id-prefix: dm
  status: backlog
  created: "2026-07-31"

tasks: []
`

func TestLoadValid(t *testing.T) {
	root, err := Load([]byte(validDoc))
	if err != nil {
		t.Fatalf("Load valid: %v", err)
	}
	if root == nil {
		t.Fatal("expected a root node")
	}
}

func TestLoadRejects(t *testing.T) {
	cases := map[string]string{
		"duplicate key":  "a: 1\na: 2\n",
		"anchor":         "a: &x 1\nb: 2\n",
		"alias":          "a: &x 1\nb: *x\n",
		"merge key":      "m:\n  <<:\n    x: 1\n",
		"unquoted date":  "d: 2026-07-31\n",
		"float tag":      "n: !!float 1.5\n",
		"boolean scalar": "flag: true\n",
		"root not map":   "- 1\n- 2\n",
		"empty document": "",
	}
	for name, src := range cases {
		if _, err := Load([]byte(src)); err == nil {
			t.Errorf("%s: expected an error, got none", name)
		}
	}
}

func TestLoadRejectsMultipleDocuments(t *testing.T) {
	src := validDoc + "---\nschema-version: 1\n"
	if _, err := Load([]byte(src)); err == nil {
		t.Fatal("expected multiple-document error")
	}
}
