package lib

import "testing"

func TestCoalesceString(t *testing.T) {
	if got := CoalesceString("", "fallback", "later"); got != "fallback" {
		t.Fatalf("CoalesceString = %q", got)
	}
	if got := CoalesceString("", ""); got != "" {
		t.Fatalf("CoalesceString empty = %q", got)
	}
}

func TestCoalesceWithPointers(t *testing.T) {
	var a *string
	b := "value"
	if got := Coalesce(a, &b); got == nil || *got != "value" {
		t.Fatalf("Coalesce = %#v", got)
	}
}

func TestCoalesceAllNilReturnsFirst(t *testing.T) {
	var a, b *string
	if got := Coalesce(a, b); got != a {
		t.Fatalf("Coalesce = %#v, want first (nil) param", got)
	}
}
