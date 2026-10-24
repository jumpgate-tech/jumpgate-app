package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

// deliverTimeout bounds one webhook attempt; deliver tries twice.
const deliverTimeout = 10 * time.Second

// deliverRetryWait is the pause before deliver's one retry; a var for tests.
var deliverRetryWait = time.Second

// queueSize is how many alert changes may wait for the webhook; more are
// dropped with a log line, so a dead webhook never holds up a poll.
const queueSize = 32

// newWebhookClient builds the webhook's client; a var for tests.
var newWebhookClient = func() *http.Client { return &http.Client{} }

// hookBody is exactly what the webhook receives.
type hookBody struct {
	Box     string     `json:"box"`
	Key     string     `json:"key"`
	Message string     `json:"message"`
	Since   time.Time  `json:"since"`
	Cleared *time.Time `json:"cleared"`
}

// errDelivery is every delivery failure: it names the status or the kind of
// error, never the URL (the webhook is a secret, and net/http's errors
// quote it).
type errDelivery string

func (e errDelivery) Error() string { return "webhook delivery failed: " + string(e) }

// deliver POSTs one alert change to the webhook at url as JSON
// {"box","key","message","since","cleared"}: https only, no redirects
// followed, 10s per attempt and one retry. Failures are logged without
// the URL.
//
// docs:covers watch-alerts
func deliver(ctx context.Context, hc *http.Client, rawURL, box string, a Alert) error {
	err := deliverOnce(ctx, hc, rawURL, box, a)
	var perm permanent
	if err != nil && !errors.As(err, &perm) && ctx.Err() == nil {
		select {
		case <-ctx.Done():
		case <-time.After(deliverRetryWait):
			err = deliverOnce(ctx, hc, rawURL, box, a)
		}
	}
	if err != nil {
		log.Printf("watch: %v (alert %s)", err, a.Key)
	}
	return err
}

// permanent marks a failure a retry cannot fix.
type permanent struct{ errDelivery }

func deliverOnce(ctx context.Context, hc *http.Client, rawURL, box string, a Alert) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return permanent{"the webhook is not an https:// URL"}
	}
	body, err := json.Marshal(hookBody{Box: box, Key: a.Key, Message: a.Message, Since: a.Since, Cleared: a.Cleared})
	if err != nil {
		return permanent{errDelivery("encoding the alert")}
	}
	ctx, cancel := context.WithTimeout(ctx, deliverTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return permanent{"the webhook URL does not parse"}
	}
	req.Header.Set("Content-Type", "application/json")
	c := *hc
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.Do(req)
	if err != nil {
		return errDelivery(errClass(err))
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errDelivery(fmt.Sprintf("status %d", resp.StatusCode))
	}
	return nil
}

// errClass names what kind of error err is, without its text.
func errClass(err error) string {
	var ne net.Error
	var dns *net.DNSError
	var op *net.OpError
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &ne) && ne.Timeout():
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.As(err, &dns):
		return "name lookup failed"
	case errors.As(err, &op):
		return "connection failed"
	default:
		return "request failed"
	}
}

// changes lists what moved between two states: each newly opened alert
// and each newly cleared one.
func changes(prev, next State) []Alert {
	type id struct {
		key   string
		since time.Time
	}
	was := map[id]bool{}
	for _, a := range prev.Open {
		was[id{a.Key, a.Since}] = true
	}
	var out []Alert
	for _, a := range next.Open {
		if !was[id{a.Key, a.Since}] {
			out = append(out, a)
		}
	}
	cleared := map[id]bool{}
	for _, a := range prev.Recent {
		cleared[id{a.Key, a.Since}] = true
	}
	for _, a := range next.Recent {
		if !cleared[id{a.Key, a.Since}] {
			out = append(out, a)
		}
	}
	return out
}

// webhook sends alert changes from a queue in its own goroutine, so a slow
// or dead webhook never delays a poll.
type webhook struct {
	q chan Alert
}

func startWebhook(ctx context.Context, rawURL, box string) *webhook {
	w := &webhook{q: make(chan Alert, queueSize)}
	hc := newWebhookClient()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case a := <-w.q:
				_ = deliver(ctx, hc, rawURL, box, a)
			}
		}
	}()
	return w
}

func (w *webhook) send(prev, next State) {
	for _, a := range changes(prev, next) {
		select {
		case w.q <- a:
		default:
			log.Printf("watch: webhook queue full; dropped alert %s", a.Key)
		}
	}
}
