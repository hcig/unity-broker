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
