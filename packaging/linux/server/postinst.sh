#!/bin/sh
# postinst for the jumpgate deb and rpm: create the service's users and
# reload systemd. It never turns the service on and never plainly restarts
# it: a restart brings the controller back locked, which stops every box it
# drives until someone unlocks it. Only a service sealed for unattended
# unlock (jumpgate vault seal) is restarted onto the new binary, and only if
# it is already running.
set -e
if [ -d /run/systemd/system ]; then
	systemd-sysusers /usr/lib/sysusers.d/jumpgate.conf
	systemctl daemon-reload
	if [ -f /etc/systemd/system/jumpgate.service.d/50-vault-seal.conf ]; then systemctl try-restart jumpgate.service; fi
fi
exit 0
