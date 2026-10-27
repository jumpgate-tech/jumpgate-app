//go:build !unix

package vault

import "os"

// checkWriteOwner does nothing: fsperm.CheckPrivateFile already refused a
// vault this user does not own.
func checkWriteOwner(string, os.FileInfo) error { return nil }
