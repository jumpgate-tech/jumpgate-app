package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// ModulePort is one public port a box module listens on.
type ModulePort struct {
	Instance string // "el-bootnode-943"
	Name     string // the module's display name: "EL bootnode"
	Port     int
	Proto    string // "tcp" | "udp"
}

// ModulePortsFromConfig lists the public ports of a target's recorded box
// modules (the controller's view, over SSH).
func ModulePortsFromConfig(mods []catalog.BoxModule) ([]ModulePort, error) {
	var out []ModulePort
	for _, b := range mods {
		u, err := catalog.RenderBoxModule(b)
		if err != nil {
			return nil, fmt.Errorf("ops: firewall: %s: %w", b.Instance(), err)
		}
		m, _ := catalog.ModuleByID(b.ID)
		for _, p := range u.Ports {
			if p.Public {
				out = append(out, ModulePort{Instance: b.Instance(), Name: m.Name, Port: p.Port, Proto: p.Proto})
			}
		}
	}
	return out, nil
}

// ModulePortsOnBox lists the public ports of the box modules installed here,
// from their root-owned sidecars (the gate's view: it has no config.json).
// A sidecar that is unreadable, not a known module, or not at its own path
// is skipped.
func ModulePortsOnBox(ctx context.Context, e executor.Executor) []ModulePort {
	res, err := e.Run(ctx, "ls "+catalog.BoxModulesConfigRoot+"/*/module.json", nil)
	if err != nil || res.ExitCode != 0 {
		return nil
	}
	var out []ModulePort
	for _, path := range strings.Fields(res.Stdout) {
		cat, err := e.Run(ctx, "cat "+shQuote(path), nil)
		if err != nil || cat.ExitCode != 0 {
			continue
		}
		var side catalog.BoxSidecar
		if json.Unmarshal([]byte(cat.Stdout), &side) != nil {
			continue
		}
		b := side.Module()
		if catalog.ValidateBoxModule(b) != nil || b.SidecarPath() != path || side.Unit != b.UnitName() {
			continue
		}
		if !side.Public || side.Port < 1 || side.Port > 65535 || (side.Proto != "tcp" && side.Proto != "udp") {
			continue
		}
		m, _ := catalog.ModuleByID(b.ID)
		out = append(out, ModulePort{Instance: b.Instance(), Name: m.Name, Port: side.Port, Proto: side.Proto})
	}
	return out
}

// ModuleFirewallItems adds a checklist item for every public module port
// ("UDP 30301 listening publicly (EL bootnode)"). It checks the bind only:
// whether the port is open in a host firewall or a cloud security group is
// not something `ss` can see, so a pass says to check that too. Like
// FirewallChecklist it only reads. With no ports it runs nothing.
func ModuleFirewallItems(ctx context.Context, e executor.Executor, ports []ModulePort) ([]CheckItem, error) {
	if len(ports) == 0 {
		return nil, nil
	}
	ss := map[string]string{}
	for proto, cmd := range map[string]string{"tcp": "ss -ltn", "udp": "ss -lun"} {
		res, err := e.Run(ctx, cmd, nil)
		if err != nil {
			return nil, fmt.Errorf("ops: firewall: %s: %w", cmd, err)
		}
		ss[proto] = res.Stdout
	}
	var items []CheckItem
	for _, p := range ports {
		it := CheckItem{
			ID:    fmt.Sprintf("module-%s-%s-%d", p.Instance, p.Proto, p.Port),
			Title: fmt.Sprintf("%s %d listening publicly (%s)", strings.ToUpper(p.Proto), p.Port, p.Name),
			Why:   "Other nodes reach the " + p.Name + " on this port from the internet; a closed or loopback-only port makes it useless.",
		}
		fix := fmt.Sprintf("ufw allow %d/%s", p.Port, p.Proto)
		switch bindState(ss[p.Proto], p.Port) {
		case "wide":
			it.Status = "pass"
			it.Detail = fmt.Sprintf("bound to a public address (%s %d). Also open it in your firewall and any cloud security group.", p.Proto, p.Port)
		case "loopback":
			it.Status, it.Detail, it.Fix = "warn", "listening on loopback only: other nodes cannot reach it", fix
		default:
			it.Status, it.Detail, it.Fix = "unknown", fmt.Sprintf("not currently listening on %s %d (the module may not be up yet)", p.Proto, p.Port), fix
		}
		items = append(items, it)
	}
	return items, nil
}
