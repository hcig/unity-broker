package lib

import "testing"

func TestSetOperations(t *testing.T) {
	s := NewSet[int]()
	if s.Has(1) {
		t.Fatal("new set should be empty")
	}
	s.Add(2)
	s.Add(5)
	s.Add(3)
	if !s.Has(5) || s.Has(1) {
		t.Fatalf("set membership incorrect: %#v", s)
	}
	if got := s.Highest(); got == nil || *got != 5 {
		t.Fatalf("Highest = %#v", got)
	}
	if got := s.Lowest(); got == nil || *got != 2 {
		t.Fatalf("Lowest = %#v", got)
	}
	s.Del(5)
	if s.Has(5) {
		t.Fatal("Del did not remove item")
	}
}

func TestSetHighAndLowOnEmptySet(t *testing.T) {
	s := NewSet[string]()
	if s.Highest() != nil || s.Lowest() != nil {
		t.Fatal("empty set should return nil extremes")
	}
}
