//go:build cgo

package buildinfo

// cgoEnabled is true when this binary was built with cgo, which links the
// build host's libc (and, for tray builds, GTK and WebKit).
const cgoEnabled = true
