package catalog

import "testing"

func TestValidateDataDir(t *testing.T) {
	ok := []string{
		"/var/lib/valve-node-app/369",
		"/mnt/nvme0/reth",
		"/data/eth",
		"/opt/jumpgate/mainnet",
		"/var/lib/jumpgate-node", // a sibling, not inside /var/lib/jumpgate
		"/var/lib/jumpgate-node/369",
		"/etc/jumpgate-node/data",
		"/usr/local/lib/jumpgate2/x",
	}
	bad := []string{
		"",                          // empty
		"data",                      // relative: rm -rf would run relative to the login dir
		"./data",                    // relative
		"/",                         // root
		"/var",                      // system tree: chown -R would re-own it
		"/var/lib",                  // system tree
		"/etc",                      // system tree
		"/home",                     // system tree
		"/var/lib/x/../..",          // not clean
		"/var/lib/x/",               // not clean (trailing slash)
		"/data/with space",          // breaks ExecStart argument splitting
		"/data/new\nline",           // injects a unit directive
		"/data/tab\there",           // control character
		"/a",                        // fewer than two components
		"/var/lib/jumpgate",         // jumpgate's own tree
		"/var/lib/jumpgate/markers", // inside it: would hand the markers dir to the service user
		"/var/lib/jumpgate/x/y",     // inside it
		"/etc/jumpgate",
		"/etc/jumpgate/data",
		"/usr/local/lib/jumpgate",
		"/usr/local/lib/jumpgate/d",
		"/usr/local/lib", // contains /usr/local/lib/jumpgate
		"/usr/local",     // contains it too
	}
	for _, p := range ok {
		if err := ValidateDataDir(p); err != nil {
			t.Errorf("ValidateDataDir(%q) = %v, want nil", p, err)
		}
	}
	for _, p := range bad {
		if err := ValidateDataDir(p); err == nil {
			t.Errorf("ValidateDataDir(%q) = nil, want an error", p)
		}
	}
}
