package guide

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every .md file here is a topic, and every topic is a file: a page added
// to the directory and left out of order would never be shown.
func TestEveryFileIsATopic(t *testing.T) {
	paths, err := filepath.Glob("*.md")
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, tp := range Embedded().Topics() {
		have[tp.Name] = true
		if tp.Title == tp.Name || tp.Body == "" {
			t.Errorf("%s: no title or body", tp.Name)
		}
	}
	for _, p := range paths {
		if name := strings.TrimSuffix(p, ".md"); !have[name] {
			t.Errorf("%s is not in the topic order", p)
		}
	}
	if len(paths) != len(have) {
		t.Errorf("%d files, %d topics", len(paths), len(have))
	}
}

// The guide is plain markdown: no raw HTML, which the TUI could not draw.
func TestNoRawHTML(t *testing.T) {
	tag := regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	for _, tp := range Embedded().Topics() {
		inCode := false
		for i, l := range strings.Split(tp.Body, "\n") {
			if strings.HasPrefix(l, "```") {
				inCode = !inCode
				continue
			}
			// Backticked text and code blocks may show a <placeholder>.
			if !inCode && tag.MatchString(regexp.MustCompile("`[^`]*`").ReplaceAllString(l, "")) {
				t.Errorf("%s.md:%d: raw HTML: %s", tp.Name, i+1, l)
			}
		}
	}
}

// Frontmatter is tolerated: its title is used and it is not drawn.
func TestFrontmatterIsSkipped(t *testing.T) {
	tp := parse("x", "---\nid: x\ntitle: \"From front\"\nsummary: s\n---\n# Heading\n\nText.\n")
	if tp.Title != "From front" || tp.Body != "# Heading\n\nText.\n" {
		t.Fatalf("%#v", tp)
	}
	if tp := parse("y", "# Heading\n"); tp.Title != "Heading" || tp.Body != "# Heading\n" {
		t.Fatalf("%#v", tp)
	}
}
