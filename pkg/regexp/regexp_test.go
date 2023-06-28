package regexp

import (
	"testing"
)

type partOfRegexp interface {
	FindStringSubmatch(s string) []string
	MatchString(s string) bool
	NumSubexp() int
}

var _ partOfRegexp = &Regexp{}

func TestMatchString(t *testing.T) {
	r := Delayed(`[0-9]+`)

	if !r.MatchString("100") {
		t.Fatalf("Should have matched")
	}

	if r.MatchString("test") {
		t.Fatalf("Should not have matched")
	}
}

func TestFindStringSubmatch(t *testing.T) {
	r := Delayed(`a=([0-9]+).*b=([0-9]+)`)

	if len(r.FindStringSubmatch("a=1,b=2")) != 3 {
		t.Fatalf("Should have matched 3")
	}

	if len(r.FindStringSubmatch("a=1")) != 0 {
		t.Fatalf("Should not have matched 0")
	}
}
