package ops

import (
	"encoding/csv"
	"strings"
	"testing"
)

// docker reads --mount as one CSV record, so a comma or quote in a path
// must be quoted. A Windows path's drive-letter colon needs nothing.
func TestBindMountQuotesAPathWithACommaOrQuote(t *testing.T) {
	for _, src := range []string{`C:\Users\dev\.valve-node-app\erpc.yaml`, `C:\Users\Ann, B\.valve-node-app\erpc.yaml`, `/home/o'neil/x.yaml`, `/Users/a "b"/x.yaml`} {
		got := bindMount(src, "/erpc.yaml")
		rec, err := csv.NewReader(strings.NewReader(got)).Read()
		if err != nil {
			t.Fatalf("%q: not one CSV record: %v", got, err)
		}
		want := []string{"type=bind", "source=" + src, "target=/erpc.yaml", "readonly"}
		if strings.Join(rec, "\x00") != strings.Join(want, "\x00") {
			t.Fatalf("%q parsed as %q, want %q", got, rec, want)
		}
	}
}

func TestERPCRunArgsMountsTheConfigFile(t *testing.T) {
	args := ERPCRunArgs(ERPCRunSpec{HostConfigPath: `C:\Users\dev\.valve-node-app\erpc.yaml`, Platform: "linux/amd64"})
	if got := valueAfter(t, args, "--mount"); got != `type=bind,source=C:\Users\dev\.valve-node-app\erpc.yaml,target=/erpc.yaml,readonly` {
		t.Fatalf("--mount %q", got)
	}
	for _, a := range args {
		if a == "-v" {
			t.Fatalf("a file bind still uses -v: %q", args)
		}
	}
}
