package ops

import (
	"context"
	"fmt"
	"strings"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// ModuleFirewallItems adds a checklist item for every public port the box's
// modules listen on ("UDP 30301 open to the internet (EL bootnode)"). Like
// FirewallChecklist it only reads: each fix is returned as text. With no
// public module port it runs nothing.
func ModuleFirewallItems(ctx context.Context, e executor.Executor, mods []catalog.BoxModule) ([]CheckItem, error) {
	type use struct {
		b    catalog.BoxModule
		name string
		p    catalog.PortUse
	}
	var uses []use
	for _, b := range mods {
		u, err := catalog.RenderBoxModule(b)
		if err != nil {
			return nil, fmt.Errorf("ops: firewall: %s: %w", b.Instance(), err)
		}
		m, _ := catalog.ModuleByID(b.ID)
		for _, p := range u.Ports {
			if p.Public {
				uses = append(uses, use{b, m.Name, p})
			}
		}
	}
	if len(uses) == 0 {
		return nil, nil
	}
	ss := map[string]string{}
	for proto, cmd := range map[string]string{"tcp": "ss -ltn", "udp": "ss -lun"} {
		res, err := e.Run(ctx, cmd, nil)
		if err != nil {
			return nil, fmt.Errorf("ops: firewall: ss: %w", err)
		}
		ss[proto] = res.Stdout
	}
	var items []CheckItem
	for _, u := range uses {
		it := CheckItem{
			ID:    "module-" + u.b.Instance() + "-" + u.p.Proto + "-open",
			Title: fmt.Sprintf("%s %d open to the internet (%s)", strings.ToUpper(u.p.Proto), u.p.Port, u.name),
			Why:   "Other nodes reach the " + u.name + " on this port from the internet; a closed or loopback-only port makes it useless.",
		}
		fix := fmt.Sprintf("ufw allow %d/%s", u.p.Port, u.p.Proto)
		switch bindState(ss[u.p.Proto], u.p.Port) {
		case "wide":
			it.Status, it.Detail = "pass", fmt.Sprintf("listening on a publicly-reachable address (%s %d)", u.p.Proto, u.p.Port)
		case "loopback":
			it.Status, it.Detail, it.Fix = "warn", "listening on loopback only: other nodes cannot reach it", fix
		default:
			it.Status, it.Detail, it.Fix = "unknown", fmt.Sprintf("not currently listening on %s %d (the module may not be up yet)", u.p.Proto, u.p.Port), fix
		}
		items = append(items, it)
	}
	return items, nil
}
