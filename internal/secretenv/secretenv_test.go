package secretenv

import (
	"reflect"
	"testing"
)

func TestIsSecret(t *testing.T) {
	for _, name := range []string{
		"JUMPGATE_ADMIN_TOKEN", "JUMPGATE_RELAY_TOKEN", "JUMPGATE_ADMIN_TOKEN_FILE", "JUMPGATE_SOME_TOKEN_V2",
		"AWS_SECRET_ACCESS_KEY", "CLIENT_SECRET", "MY_SECRET", "JUMPGATE_SECRET",
	} {
		if !IsSecret(name) {
			t.Errorf("%s is not treated as a secret", name)
		}
	}
	for _, name := range []string{
		"PATH", "HOME", "JUMPGATE_RELAY_BIND", "JUMPGATE_METER", "OP_SESSION_my", "OP_SERVICE_ACCOUNT_TOKEN", "SSH_AUTH_SOCK", "TOKENIZER",
	} {
		if IsSecret(name) {
			t.Errorf("%s is treated as a secret", name)
		}
	}
}

func TestScrubDropsSecretsAndKeepsTheRest(t *testing.T) {
	in := []string{"PATH=/bin", "JUMPGATE_ADMIN_TOKEN=a", "HOME=/h", "CLIENT_SECRET=s", "JUMPGATE_RELAY_TOKEN=r", "OP_SESSION_x=keep"}
	got := Scrub(in)
	want := []string{"PATH=/bin", "HOME=/h", "OP_SESSION_x=keep"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scrub = %v, want %v", got, want)
	}
	if in[1] != "JUMPGATE_ADMIN_TOKEN=a" {
		t.Fatal("Scrub modified its input")
	}
}
