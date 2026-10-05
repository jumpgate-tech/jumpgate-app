#!/bin/sh
# packaging/macos/jumpgate.command — run by the terminal Jumpgate Terminal.app
# opened. JUMPGATE_LAUNCHER tells jumpgate it owns this window, so an early
# error waits for Enter instead of vanishing.
export JUMPGATE_LAUNCHER=1
exec "$(dirname "$0")/jumpgate"
