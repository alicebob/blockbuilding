package main

import (
	"testing"
)

func TestMatch(t *testing.T) {
	for _, tc := range []struct {
		url, domain string
		want        bool
	}{
		{"http://foobar.com", "foobar.com", true},
		{"http://www.foobar.com", "foobar.com", false},
		{"http://www.foobar.com", ".foobar.com", true},
		{"http://www.foofoobar.com", "foobar.com", false},
		{"http://www.foofoobar.com", ".foobar.com", false},
	} {
		if have, want := matchesDomain(tc.url, tc.domain), tc.want; have != want {
			t.Errorf("%q -> %q have: %v, want %v", tc.url, tc.domain, have, want)
		}
	}
}
