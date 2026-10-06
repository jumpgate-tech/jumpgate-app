//go:build !linux

package server

import "net/http"

// defaultPeerUID is nil off Linux: there is no cheap, reliable way to read
// the owner of a loopback TCP peer, so the login path relies on the code
// never leaving its owner-only file.
var defaultPeerUID func(*http.Request) (int, peerVerdict)
