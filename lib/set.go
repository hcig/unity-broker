package lib

import "cmp"

type Set[K cmp.Ordered] map[K]bool

func NewSet[K cmp.Ordered]() Set[K] {
	return Set[K]{}
}

func (s Set[K]) Has(needle K) bool {
	_, found := s[needle]
	return found
}

func (s Set[K]) Add(needle K) {
	s[needle] = true
}

func (s Set[K]) Del(needle K) {
	delete(s, needle)
}

func (s Set[K]) Highest() *K {
	var m *K = nil
	for k, _ := range s {
		if m == nil || *m < k {
			m = &k
		}
	}
	return m
}

func (s Set[K]) Lowest() *K {
	var m *K = nil
	for k, _ := range s {
		if m == nil || *m > k {
			m = &k
		}
	}
	return m
}
