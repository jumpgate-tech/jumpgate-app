package jobs

import (
	"context"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// scriptExec answers each command by the longest scripted substring it
// contains (as internal/setup's fake does); an unscripted command succeeds
// with no output. Files written through it are kept in memory.
type scriptExec struct {
	mu      sync.Mutex
	scripts map[string][]executor.Result // a list answers successive calls; the last repeats
	calls   []string
	files   map[string][]byte
}

func newScript() *scriptExec {
	return &scriptExec{scripts: map[string][]executor.Result{}, files: map[string][]byte{}}
}

func (s *scriptExec) on(substr string, rs ...executor.Result) *scriptExec {
	s.mu.Lock()
	s.scripts[substr] = rs
	s.mu.Unlock()
	return s
}

func (s *scriptExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, cmd)
	var keys []string
	for k := range s.scripts {
		if strings.Contains(cmd, k) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return executor.Result{}, nil
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	rs := s.scripts[keys[0]]
	r := rs[0]
	if len(rs) > 1 {
		s.scripts[keys[0]] = rs[1:]
	}
	return r, nil
}

func (s *scriptExec) WriteFile(_ context.Context, path string, b []byte, _ os.FileMode) error {
	s.mu.Lock()
	s.files[path] = b
	s.mu.Unlock()
	return nil
}

func (s *scriptExec) ReadFile(_ context.Context, path string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok := s.files[path]; ok {
		return b, nil
	}
	return nil, os.ErrNotExist
}

func (s *scriptExec) Close() error { return nil }

func (s *scriptExec) ran(substr string) bool { return s.index(substr) >= 0 }

func (s *scriptExec) index(substr string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.calls {
		if strings.Contains(c, substr) {
			return i
		}
	}
	return -1
}

var ok0 = executor.Result{}
var fail1 = executor.Result{ExitCode: 1}

func idOK(uid string) executor.Result { return executor.Result{Stdout: uid + "\n"} }

func stdout(s string) executor.Result { return executor.Result{Stdout: s} }

// legacyBox scripts a box set up before sub-project 2: the legacy account
// exists, the new one does not, and nothing sits at the new data root or
// under the new unit names.
func legacyBox() *scriptExec {
	return newScript().
		on("id -u valve-node-app", idOK("998")).
		on("id -u jumpgate-node", fail1).
		on("getent group valve-node-app", stdout("valve-node-app:x:998:\n")).
		on("getent group jumpgate-node", fail1).
		on("test -e '/var/lib/jumpgate-node'", fail1).
		on("systemctl is-enabled jumpgate-node-", fail1)
}

// migratedBox scripts a box node.migrate-names finished on: only the new
// account and group exist, and no legacy unit file is left.
func migratedBox() *scriptExec {
	return newScript().
		on("id -u valve-node-app", fail1).
		on("id -u jumpgate-node", idOK("998")).
		on("getent group valve-node-app", fail1).
		on("getent group jumpgate-node", stdout("jumpgate-node:x:998:\n")).
		on("test -e '/etc/systemd/system/valve-node-app-", fail1)
}
