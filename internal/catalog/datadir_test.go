package catalog

import "testing"

func TestValidateDataDir(t *testing.T) {
	ok := []string{
		"/var/lib/valve-node-app/369",
		"/mnt/nvme0/reth",
		"/data/eth",
		"/opt/jumpgate/mainnet",
	}
	bad := []string{
		"",                 // empty
		"data",             // relative: rm -rf would run relative to the login dir
		"./data",           // relative
		"/",                // root
		"/var",             // system tree: chown -R would re-own it
		"/var/lib",         // system tree
		"/etc",             // system tree
		"/home",            // system tree
		"/var/lib/x/../..", // not clean
		"/var/lib/x/",      // not clean (trailing slash)
		"/data/with space", // breaks ExecStart argument splitting
		"/data/new\nline",  // injects a unit directive
		"/data/tab\there",  // control character
		"/a",               // fewer than two components
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
