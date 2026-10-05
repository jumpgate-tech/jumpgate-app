// Package testhook carries apiclient's test-only constructor to apiclienttest.
// It is internal to apiclient, so nothing outside that subtree can import it.
package testhook

// NewHTTP is set by apiclient's init to its unexported newHTTP. It returns
// *apiclient.Client as any, because this package cannot import apiclient.
var NewHTTP func(baseURL, token, version string) any
