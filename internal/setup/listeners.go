package setup

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// localDialTimeout bounds one loopback connect in probeListeners.
var localDialTimeout = 500 * time.Millisecond

// probeListeners reports what already listens on port on the target, or ""
// when nothing does. On the local machine it connects to the loopback
// addresses directly (spec D32), because ss, netstat and lsof sit behind a
// shell. A listener on a wildcard or loopback address accepts that connect,
// and those are the only listeners a 127.0.0.1 publish collides with. For a
// wildcard publish, docker still fails loudly on a real collision.
// Anywhere else it runs listenerProbe as before.
func probeListeners(ctx context.Context, e executor.Executor, port int) (string, error) {
	h, ok := e.(executor.LocalHost)
	if !ok {
		res, err := e.Run(ctx, fmt.Sprintf(listenerProbe, port), nil)
		if err != nil {
			return "", err
		}
		if res.ExitCode == 0 {
			return strings.TrimSpace(res.Stdout), nil
		}
		return "", nil
	}
	for _, host := range []string{"127.0.0.1", "::1"} {
		addr := net.JoinHostPort(host, strconv.Itoa(port))
		dctx, cancel := context.WithTimeout(ctx, localDialTimeout)
		c, err := h.DialContext(dctx, "tcp", addr)
		cancel()
		if err == nil {
			c.Close()
			return "something accepts connections on " + addr, nil
		}
	}
	return "", nil
}
