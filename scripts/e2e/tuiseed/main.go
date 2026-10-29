// Command tuiseed prints the shell script scripts/e2e-tui.sh runs as root on
// a box from scripts/e2e/Dockerfile: a node set up before the rename (legacy
// names, the recipe of internal/bootstrap's seedLegacyNode), running on the
// image's stub clients, and one process of its account that ignores SIGTERM,
// so node.migrate-names asks what to do about it.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/valve-tech/jumpgate/internal/bootstrap"
	"github.com/valve-tech/jumpgate/internal/catalog"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "tuiseed:", err)
		os.Exit(1)
	}
}

// heredoc writes body to path with a quoted heredoc, so nothing expands.
func heredoc(path, body string) string {
	return fmt.Sprintf("mkdir -p %s\ncat > %s <<'JUMPGATE_SEED_EOF'\n%s\nJUMPGATE_SEED_EOF\n", dir(path), path, strings.TrimRight(body, "\n"))
}

func dir(p string) string { return p[:strings.LastIndex(p, "/")] }

func run() error {
	legacy := catalog.LegacyNames
	w := catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
		DataDir: legacy.DataRoot + "/369", JWTPath: legacy.DataRoot + "/369/jwt.hex"}
	execUnit, beaconUnit, err := catalog.RenderUnits(w)
	if err != nil {
		return err
	}
	node, err := json.Marshal(w)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("useradd --system --user-group --home-dir " + legacy.DataRoot + " --no-create-home --shell /usr/sbin/nologin " + legacy.User + "\n")
	b.WriteString("mkdir -p " + w.DataDir + "/db && head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \\n' > " + w.JWTPath + "\n")
	b.WriteString("chown -R " + legacy.User + ":" + legacy.Group + " " + legacy.DataRoot + "\n")
	b.WriteString(heredoc(catalog.UnitPath(legacy.ExecUnit()), execUnit))
	b.WriteString(heredoc(catalog.UnitPath(legacy.BeaconUnit()), beaconUnit))
	b.WriteString(heredoc(bootstrap.NodePath, string(node)))
	b.WriteString("chmod 600 " + bootstrap.NodePath + "\n")
	b.WriteString(catalog.SnapshotMarkerCommand(w) + "\n")
	b.WriteString(catalog.OwnerMarkerCommand(w) + "\n")
	b.WriteString("systemctl daemon-reload && systemctl enable --now " + strings.Join(legacy.Units(), " ") + "\n")
	b.WriteString(`systemd-run --unit=e2e-busy --uid=` + legacy.User + ` --quiet /bin/sh -c "trap '' TERM; while :; do sleep 1; done"` + "\n")
	_, err = os.Stdout.WriteString(b.String())
	return err
}
