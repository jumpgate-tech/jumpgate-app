package bootstrap

import "errors"

// ErrLocalUnsupported refuses --local off Linux: it pairs the machine the
// controller runs on, and the agent runs only on Linux (I-10).
var ErrLocalUnsupported = errors.New("--local pairs this machine, and the jumpgate agent runs only on Linux; pair a Linux box with --ssh instead")

// LocalSupported reports whether a controller on goos can pair itself.
func LocalSupported(goos string) error {
	if goos == "linux" {
		return nil
	}
	return ErrLocalUnsupported
}
