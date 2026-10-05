package api

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/valve-tech/jumpgate/internal/intent"
)

var snake = regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)

func TestEveryCodeIsSnakeCaseAndRegistered(t *testing.T) {
	if len(Codes()) < 30 {
		t.Fatalf("registry has %d codes, want the full set", len(Codes()))
	}
	for _, c := range Codes() {
		if !snake.MatchString(string(c)) {
			t.Errorf("code %q is not snake_case", c)
		}
		if e := c.Exit(); e < ExitFailed || e > ExitSecurity {
			t.Errorf("code %q has exit class %d", c, e)
		}
	}
}

// The CLI's exit statuses are a published table (README "Exit codes").
func TestExitClassesMatchTheCLITable(t *testing.T) {
	for c, want := range map[Code]int{
		CodeUnreachable: ExitUnreachable, CodeBadReceipt: ExitSecurity, CodeHostKey: ExitSecurity,
		CodeUnknownHost: ExitSecurity, CodeNoControllerKey: ExitUsage, CodeNotPaired: ExitUsage,
		CodeAgentHTTP: ExitFailed, CodeRejected: ExitFailed, CodeTargetNotFound: ExitFailed,
		CodeControllerKeyMismatch: ExitSecurity,
	} {
		if got := c.Exit(); got != want {
			t.Errorf("%s.Exit() = %d, want %d", c, got, want)
		}
	}
	if Code("made_up").Exit() != ExitFailed {
		t.Error("an unknown code must exit 1")
	}
}

func TestCodeForStatus(t *testing.T) {
	for status, want := range map[int]Code{
		400: CodeBadRequest, 401: CodeUnauthorized, 403: CodeForbidden, 404: CodeNotFound, 409: CodeConflict,
		413: CodeTooLarge, 500: CodeInternal, 501: CodeNotImplemented, 502: CodeUpstream, 503: CodeUnavailable,
		504: CodeTimeout, 418: CodeBadRequest, 599: CodeInternal,
	} {
		if got := CodeForStatus(status); got != want {
			t.Errorf("CodeForStatus(%d) = %s, want %s", status, got, want)
		}
	}
}

func TestDecodeReadsJSONAndToleratesPlainText(t *testing.T) {
	e := Decode(http.StatusConflict, []byte(`{"error":"no","hint":"h","code":"unknown_host","host":"h:22","fingerprint":"SHA256:x"}`))
	if e.Status != 409 || e.Message != "no" || e.Code != CodeUnknownHost || e.Host != "h:22" || e.Fingerprint != "SHA256:x" {
		t.Fatalf("decoded %+v", e)
	}
	p := Decode(http.StatusUnauthorized, []byte("unauthorized\n"))
	if p.Message != "unauthorized" || p.Code != CodeUnauthorized || p.Hint == "" {
		t.Fatalf("plain text decoded as %+v", p)
	}
	empty := Decode(http.StatusBadGateway, nil)
	if empty.Message != "Bad Gateway" || empty.Code != CodeUpstream {
		t.Fatalf("empty body decoded as %+v", empty)
	}
}

func TestEveryRejectionReasonHasAHint(t *testing.T) {
	for _, r := range []string{
		intent.ReasonWrongAgent, intent.ReasonExpired, intent.ReasonClockSkew, intent.ReasonBadSignature,
		intent.ReasonUnauthorizedKind, intent.ReasonUnknownKind, intent.ReasonStaleSeq, intent.ReasonReplayedNonce,
		intent.ReasonBusy, intent.ReasonInvalidPayload, intent.ReasonValidation, intent.ReasonNotSetUp,
		intent.ReasonReplayState,
	} {
		if RejectionHint(r) == "" {
			t.Errorf("no hint for rejection %q", r)
		}
	}
}

func TestErrorStringNamesTheCode(t *testing.T) {
	e := &Error{Message: "box is down", Code: CodeUnreachable}
	if e.Error() != "box is down (unreachable)" {
		t.Fatalf("Error() = %q", e.Error())
	}
}
