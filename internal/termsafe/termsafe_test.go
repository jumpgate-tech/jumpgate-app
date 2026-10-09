package termsafe

import "testing"

func TestTextAndLine(t *testing.T) {
	if got := Text("a\x1b]52;c;QQ==\x07b\x1b[2Jc\r\x00\td\n"); got != "abc\td\n" {
		t.Errorf("Text = %q", got)
	}
	if got := Line("a\tb\nc\r"); got != "a b c" {
		t.Errorf("Line = %q", got)
	}
}
