package tui

import (
	"context"
	"testing"
)

func TestVisitBeginEndInvalidate(t *testing.T) {
	var v visit
	v.begin(context.Background())
	first, ctx1 := v.gen, v.ctx
	if !v.current(first) {
		t.Fatal("the new visit's gen is not current")
	}
	v.begin(context.Background())
	if ctx1.Err() == nil || v.current(first) {
		t.Fatalf("a new visit must cancel the old and make its gen stale (err %v)", ctx1.Err())
	}
	gen := v.gen
	v.end()
	if v.ctx.Err() == nil || !v.current(gen) {
		t.Fatal("end cancels the work but leaves in-flight replies current")
	}
	v.invalidate()
	if v.current(gen) {
		t.Fatal("invalidate must make in-flight replies stale")
	}
	v.open(41, context.Background())
	if !v.current(41) {
		t.Fatal("open uses the caller's gen")
	}
}
