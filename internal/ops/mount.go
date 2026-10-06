package ops

import (
	"encoding/csv"
	"strings"
)

// bindMount renders a read-only file bind as a --mount value (spec D33).
// --mount, not -v, for two reasons. First, a Windows source path's drive
// letter (C:\…) is a colon that -v has to guess about. Second, --mount
// refuses a source that does not exist, where -v quietly bind-mounts an
// empty directory in its place. On colima that produced the crash loop the
// gap analysis saw live ("read /erpc.yaml: is a directory"). docker parses
// the value as one CSV record, so a comma or a quote in a path is quoted.
func bindMount(src, dst string) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"type=bind", "source=" + src, "target=" + dst, "readonly"})
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n")
}
