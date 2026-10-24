package monitor

import "testing"

func TestParseMsgboardStatus(t *testing.T) {
	const res = `{"enabled":true,"count":"0x4d2","size":"0x2f5000","headBlock":"0x10"}`
	for _, body := range []string{res, `{"jsonrpc":"2.0","id":1,"result":` + res + `}`} {
		st, err := parseMsgboardStatus(body)
		if err != nil {
			t.Fatalf("%s: %v", body, err)
		}
		if !st.Served || !st.Accepting || st.Count != 1234 || st.Size != 3100672 || st.HeadBlock != 16 {
			t.Fatalf("%s: %+v", body, st)
		}
	}
	if _, err := parseMsgboardStatus(`{"enabled":true,"count":"0xzz","size":"0x0","headBlock":"0x0"}`); err == nil {
		t.Fatal("bad hex parsed")
	}
	if _, err := parseMsgboardStatus(`{"enabled":true,"count":"12","size":"0x0","headBlock":"0x0"}`); err == nil {
		t.Fatal("hex without 0x parsed")
	}
	if nodeDecoders["msgboard"] == nil {
		t.Fatal("parseMsgboardStatus is not registered")
	}
}
