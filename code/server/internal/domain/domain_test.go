package domain

import "testing"

func TestParse(t *testing.T) {
	p := Parse([]string{"HTTPS://Example.COM/path, example.com.; bad_domain; valid.id"})
	if len(p.Domains) != 2 || p.Domains[0] != "example.com" || p.Duplicates != 1 || len(p.Invalid) != 1 {
		t.Fatalf("unexpected parse: %#v", p)
	}
}
func TestRejectInvalid(t *testing.T) {
	for _, s := range []string{"-bad.com", "bad-.com", "a..com", "localhost", "bad.123"} {
		if _, ok := Normalize(s); ok {
			t.Errorf("accepted %q", s)
		}
	}
}
