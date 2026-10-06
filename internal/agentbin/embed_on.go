//go:build embedagents

package agentbin

import "embed"

// Release builds run scripts/build-agents.sh into ./embedded with GZIP=1
// before compiling, so these files exist exactly when the tag is set.
//
//go:embed embedded/jumpgate-linux-amd64.gz embedded/jumpgate-linux-arm64.gz embedded/SHA256SUMS
var embeddedFiles embed.FS

func init() { embeddedFS = embeddedFiles }
