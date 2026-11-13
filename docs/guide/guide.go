// Package guide is the user guide, one markdown file per topic, embedded so
// the TUI can show it offline. The files are plain markdown (headings,
// paragraphs, lists, fenced code, inline code, emphasis and links; no HTML)
// and are the one source for the guide wherever it is shown.
package guide

import (
	"embed"
	"strings"
)

//go:embed *.md
var files embed.FS

// order is the reading order. A file left out of it, or a name here with no
// file, fails the package's tests.
var order = []string{"install", "getting-started", "tui", "commands", "modules", "jobs", "vault", "upgrading", "security"}

// Topic is one guide page.
type Topic struct {
	Name  string // the file name without .md, as typed in :help NAME
	Title string // the frontmatter's title, else the first "# " heading
	Body  string // the markdown, without any frontmatter
}

// Guide is where the topics come from. The TUI reads only this, so another
// source later (steps grouped into pathways, say) changes the loader, not
// the screens.
type Guide interface {
	Topics() []Topic
}

// Embedded is the guide built into the binary: the flat files here.
func Embedded() Guide { return flat{} }

type flat struct{}

// Topics is every topic in reading order.
func (flat) Topics() []Topic {
	out := make([]Topic, 0, len(order))
	for _, name := range order {
		b, err := files.ReadFile(name + ".md")
		if err != nil {
			panic("guide: " + err.Error()) // embedded at build time; the tests catch a missing file
		}
		out = append(out, parse(name, string(b)))
	}
	return out
}

// parse splits off a leading YAML frontmatter block (between "---" lines),
// taking only its title: nothing else in it is used yet.
func parse(name, src string) Topic {
	t := Topic{Name: name, Body: src}
	if rest, ok := strings.CutPrefix(src, "---\n"); ok {
		if front, body, ok := strings.Cut(rest, "\n---\n"); ok {
			t.Body = body
			for _, l := range strings.Split(front, "\n") {
				if v, ok := strings.CutPrefix(l, "title:"); ok {
					t.Title = strings.Trim(strings.TrimSpace(v), `"'`)
				}
			}
		}
	}
	if t.Title == "" {
		t.Title = name
		for _, l := range strings.Split(t.Body, "\n") {
			if h, ok := strings.CutPrefix(l, "# "); ok {
				t.Title = strings.TrimSpace(h)
				break
			}
		}
	}
	return t
}
