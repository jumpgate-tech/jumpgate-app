package executor

import "time"

// AgentAvailable reports whether an ssh-agent answers here, so an SSH target
// with no key file can still authenticate. On Windows that includes the
// OpenSSH agent service's named pipe (I-5).
func AgentAvailable() bool {
	c := dialAgent(time.Now().Add(2 * time.Second))
	if c == nil {
		return false
	}
	c.Close()
	return true
}
