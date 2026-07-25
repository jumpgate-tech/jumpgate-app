package executor

import (
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"testing"
)

// The tests in this file are the regression guard for a bug that only bites on
// a Windows control plane: remote paths built with the LOCAL separator. They
// are written so they do not depend on the host's separator — the assertions
// are on the exact strings handed to the target's shell, which must be POSIX
// on every control-plane OS. TestRemotePathFilesDoNotImportFilepath is the one
// that fails here on darwin if the bug is reintroduced.

func TestRemoteDir(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "jwt under the per-chain data dir",
			in:   "/var/lib/valve-node-app/369/jwt.hex",
			want: "/var/lib/valve-node-app/369",
		},
		{
			name: "systemd unit path",
			in:   "/etc/systemd/system/valve-node-app-exec.service",
			want: "/etc/systemd/system",
		},
		{
			name: "file directly under root",
			in:   "/jwt.hex",
			want: "/",
		},
		{
			name: "relative path",
			in:   "foo/bar.txt",
			want: "foo",
		},
		{
			name: "no directory component",
			in:   "bar.txt",
			want: ".",
		},
		{
			// A backslash is an ordinary filename character on Linux and must
			// stay one: filepath.Dir on Windows would split here (and rewrite
			// every other separator to a backslash), producing a path the
			// target has never heard of.
			name: "backslash is not a separator on the target",
			in:   `/var/lib/odd\name/file.txt`,
			want: `/var/lib/odd\name`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := remoteDir(tt.in); got != tt.want {
				t.Errorf("remoteDir(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestWriteFileCmd_BuildsPOSIXRemotePaths(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		content     []byte
		mode        fs.FileMode
		wantMkdir   string
		wantTargets string
		wantChmod   string
	}{
		{
			name:        "jwt secret",
			path:        "/var/lib/valve-node-app/369/jwt.hex",
			content:     []byte("0xdeadbeef"),
			mode:        0o600,
			wantMkdir:   "mkdir -p '/var/lib/valve-node-app/369'",
			wantTargets: "> '/var/lib/valve-node-app/369/jwt.hex'",
			wantChmod:   "chmod 600 '/var/lib/valve-node-app/369/jwt.hex'",
		},
		{
			name:        "systemd unit",
			path:        "/etc/systemd/system/valve-node-app-exec.service",
			content:     []byte("[Unit]\n"),
			mode:        0o644,
			wantMkdir:   "mkdir -p '/etc/systemd/system'",
			wantTargets: "> '/etc/systemd/system/valve-node-app-exec.service'",
			wantChmod:   "chmod 644 '/etc/systemd/system/valve-node-app-exec.service'",
		},
		{
			name:        "deeply nested config",
			path:        "/opt/valve/etc/clients/lighthouse/config.toml",
			content:     nil,
			mode:        0o640,
			wantMkdir:   "mkdir -p '/opt/valve/etc/clients/lighthouse'",
			wantTargets: "> '/opt/valve/etc/clients/lighthouse/config.toml'",
			wantChmod:   "chmod 640 '/opt/valve/etc/clients/lighthouse/config.toml'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := writeFileCmd(tt.path, tt.content, tt.mode)

			for _, want := range []string{tt.wantMkdir, tt.wantTargets, tt.wantChmod} {
				if !strings.Contains(cmd, want) {
					t.Errorf("writeFileCmd(%q) = %q\nwant it to contain %q", tt.path, cmd, want)
				}
			}
			// The command is interpreted by the target's /bin/sh. A single
			// backslash anywhere in it means a local separator leaked in.
			if strings.Contains(cmd, `\`) {
				t.Errorf("writeFileCmd(%q) contains a backslash — remote paths must be POSIX: %q", tt.path, cmd)
			}
		})
	}
}

func TestReadFileCmd_PassesRemotePathThroughVerbatim(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "jwt secret",
			path: "/var/lib/valve-node-app/369/jwt.hex",
			want: "base64 < '/var/lib/valve-node-app/369/jwt.hex'",
		},
		{
			name: "path containing a single quote is quoted safely",
			path: "/var/lib/it's/jwt.hex",
			want: `base64 < '/var/lib/it'"'"'s/jwt.hex'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readFileCmd(tt.path); got != tt.want {
				t.Errorf("readFileCmd(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestRemotePathFilesDoNotImportFilepath is a source-level guard. The two path
// styles cannot be told apart by a test running on darwin — filepath.Dir and
// path.Dir agree on every POSIX host — so the only way to catch a regression
// here, on this machine, is to assert the structural rule directly: the files
// that build paths for the remote target must not import "path/filepath" at
// all. Local-filesystem code (hostkey.go, local.go) is intentionally excluded;
// filepath is the right call there.
func TestRemotePathFilesDoNotImportFilepath(t *testing.T) {
	remoteOnlyFiles := []string{"ssh.go", "remotepath.go"}

	for _, name := range remoteOnlyFiles {
		t.Run(name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
			for _, imp := range f.Imports {
				if imp.Path.Value == `"path/filepath"` {
					t.Errorf("%s imports \"path/filepath\", but it builds paths for the remote Linux target: use \"path\" (see remoteDir in remotepath.go), or move the local-filesystem code to hostkey.go", name)
				}
			}
		})
	}
}
