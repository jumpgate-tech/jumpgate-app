// Package apiclienttest builds apiclient clients for an in-process test
// server. Its functions take a testing.TB so they are not mistaken for a
// production constructor: a real client comes from apiclient.Connect or
// apiclient.New, whose server version is the one server.json published.
package apiclienttest

import (
	"testing"

	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/apiclient/internal/testhook"
	"github.com/valve-tech/jumpgate/internal/buildinfo"
)

// NewHTTP is a client for a test server at baseURL. The server is in-process,
// so it is this build: the client reports this binary's version and no skew.
func NewHTTP(t testing.TB, baseURL, token string) *apiclient.Client {
	t.Helper()
	return NewHTTPVersion(t, baseURL, token, buildinfo.Version())
}

// NewHTTPVersion is NewHTTP for a server that reports version, to test what a
// client does against another jumpgate build.
func NewHTTPVersion(t testing.TB, baseURL, token, version string) *apiclient.Client {
	t.Helper()
	return testhook.NewHTTP(baseURL, token, version).(*apiclient.Client)
}
