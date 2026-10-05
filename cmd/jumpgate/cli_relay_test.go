package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// poisonedHome is a HOME whose controller state cannot be used: config.json
// is not JSON, a controller key file sits beside it, a legacy
// ~/.valve-node-app waits to be migrated, and ~/.jumpgate is mode 000 so any
// attempt to read config, take the server lock or make the run directory
// fails. It returns HOME and a snapshot function of what is in it.
func poisonedHome(t *testing.T) (string, func() string) {
	t.Helper()
	home, err := os.MkdirTemp("/tmp", "jgpoison")
	if err != nil {
		t.Fatal(err)
	}
	jg := filepath.Join(home, ".jumpgate")
	legacy := filepath.Join(home, ".valve-node-app")
	for _, d := range []string{filepath.Join(jg, "keys"), legacy} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p, s string) {
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(jg, "config.json"), `POISON{"controller":{"keyStore":"file","keyRef":"`+filepath.Join(jg, "keys", "controller.key")+`"`)
	write(filepath.Join(jg, "keys", "controller.key"), strings.Repeat("ab", 32)+"\n")
	write(filepath.Join(legacy, "config.json"), `{"targets":[]}`)
	if err := os.Chmod(jg, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(jg, 0o700)
		os.RemoveAll(home)
	})
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	snapshot := func() string {
		_ = os.Chmod(jg, 0o700)
		defer os.Chmod(jg, 0)
		var lines []string
		_ = filepath.Walk(home, func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(home, p)
			lines = append(lines, rel+" "+fi.Mode().String()+" "+fi.ModTime().String())
			return nil
		})
		sort.Strings(lines)
		return strings.Join(lines, "\n")
	}
	return home, snapshot
}

// D2/F3: `jumpgate relay` runs the metered data plane alone. It never opens
// config.json, the controller key or an executor, and it never migrates or
// creates controller state: the internet-facing proxy must hold no authority
// over the fleet. Its settings come only from flags and the environment.
func TestRelaySubcommandNeverTouchesControllerState(t *testing.T) {
	home, snapshot := poisonedHome(t)
	before := snapshot()

	// main's startup migration is skipped for relay, as it is for agent.
	if err := migrateOnStartup([]string{"jumpgate", "relay"}, io.Discard); err != nil {
		t.Fatalf("startup migration ran for relay: %v", err)
	}

	bind := freeAddr(t)
	env := map[string]string{
		"JUMPGATE_RELAY_BIND":     bind,
		"JUMPGATE_BILLING_SOCKET": filepath.Join(home, "billing.sock"),
		"JUMPGATE_RELAY_TOKEN":    "relay-token",
		"JUMPGATE_METER":          "true",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan int, 1)
	var log strings.Builder
	go func() {
		done <- relayMain(ctx, []string{"--erpc-url", "http://127.0.0.1:1"}, func(k string) string { return env[k] }, &log)
	}()

	// It serves: an unknown key on the data plane gets the relay's own answer.
	deadline := time.Now().Add(5 * time.Second)
	var res *http.Response
	for {
		var err error
		res, err = http.Get("http://" + bind + "/rpc/jg_nosuchkey/evm/369")
		if err == nil {
			break
		}
		select {
		case code := <-done:
			t.Fatalf("relay exited %d before serving: %s", code, log.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("relay never served on %s: %v", bind, err)
		}
		time.Sleep(25 * time.Millisecond)
	}
	res.Body.Close()
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		t.Fatalf("data plane answered %d for an unknown key", res.StatusCode)
	}

	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("relay exit %d: %s", code, log.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("relay did not stop")
	}
	if after := snapshot(); after != before {
		t.Fatalf("relay changed controller state in HOME:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestRelaySubcommandNeedsABind(t *testing.T) {
	if code := relayMain(context.Background(), nil, func(k string) string {
		if k == "JUMPGATE_RELAY_TOKEN" {
			return "t"
		}
		return ""
	}, io.Discard); code != exitCode("usage") {
		t.Fatalf("exit %d, want usage", code)
	}
}

// A half-configured relay is refused, as in the controller.
func TestRelaySubcommandRefusesAHalfConfiguredRelay(t *testing.T) {
	var log strings.Builder
	code := relayMain(context.Background(), []string{"--relay-bind", freeAddr(t)}, func(string) string { return "" }, &log)
	if code == 0 || !strings.Contains(log.String(), "billing socket") {
		t.Fatalf("exit %d, log %q", code, log.String())
	}
}

func TestRelayIsASubcommand(t *testing.T) {
	if _, ok := subcommands["relay"]; !ok {
		t.Fatal("no relay subcommand")
	}
}

func TestRelayUnsetsItsToken(t *testing.T) {
	t.Setenv("JUMPGATE_RELAY_TOKEN", "rt")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // build, then stop at once
	_ = relayMain(ctx, []string{"--relay-bind", freeAddr(t), "--billing-socket", "/tmp/jg-none.sock"}, os.Getenv, io.Discard)
	if _, set := os.LookupEnv("JUMPGATE_RELAY_TOKEN"); set {
		t.Fatal("JUMPGATE_RELAY_TOKEN is still in the environment")
	}
}
