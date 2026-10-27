//go:build !unix

package vault

import "os"

// keepFileOwner does nothing: on Windows the temp file already carries the
// owner-only DACL from fsperm.
func keepFileOwner(*os.File, os.FileInfo) error { return nil }
