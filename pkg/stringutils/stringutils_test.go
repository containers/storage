package stringutils

import "testing"

func testLengthHelper(t *testing.T, generator func(int) string) {
	expectedLength := 20
	s := generator(expectedLength)
	if len(s) != expectedLength {
		t.Fatalf("Length of %s was %d but expected length %d", s, len(s), expectedLength)
	}
}

func testUniquenessHelper(t *testing.T, generator func(int) string) {
	repeats := 25
	set := make(map[string]struct{}, repeats)
	for range repeats {
		str := generator(64)
		if len(str) != 64 {
			t.Fatalf("Id returned is incorrect: %s", str)
		}
		if _, ok := set[str]; ok {
			t.Fatalf("Random number is repeated")
		}
		set[str] = struct{}{}
	}
}

func isASCII(s string) bool {
	for _, c := range s {
		if c > 127 {
			return false
		}
	}
	return true
}

func TestGenerateRandomAlphaOnlyStringLength(t *testing.T) {
	testLengthHelper(t, GenerateRandomAlphaOnlyString)
}

func TestGenerateRandomAlphaOnlyStringUniqueness(t *testing.T) {
	testUniquenessHelper(t, GenerateRandomAlphaOnlyString)
}

func TestGenerateRandomAsciiStringLength(t *testing.T) {
	testLengthHelper(t, GenerateRandomASCIIString)
}

func TestGenerateRandomAsciiStringUniqueness(t *testing.T) {
	testUniquenessHelper(t, GenerateRandomASCIIString)
}

func TestGenerateRandomAsciiStringIsAscii(t *testing.T) {
	str := GenerateRandomASCIIString(64)
	if !isASCII(str) {
		t.Fatalf("%s contained non-ascii characters", str)
	}
}

const ststring = "t🐳ststring"

func TestEllipsis(t *testing.T) {
	newstr := Ellipsis(ststring, 3)
	if newstr != "t🐳s" {
		t.Fatalf("Expected t🐳s, got %s", newstr)
	}
	newstr = Ellipsis(ststring, 8)
	if newstr != "t🐳sts..." {
		t.Fatalf("Expected tests..., got %s", newstr)
	}
	newstr = Ellipsis(ststring, 20)
	if newstr != ststring {
		t.Fatalf("Expected %s, got %s", ststring, newstr)
	}
}

func TestTruncate(t *testing.T) {
	newstr := Truncate(ststring, 4)
	if newstr != "t🐳st" {
		t.Fatalf("Expected t🐳st, got %s", newstr)
	}
	newstr = Truncate(ststring, 20)
	if newstr != ststring {
		t.Fatalf("Expected t🐳ststring, got %s", newstr)
	}
}

func TestInSlice(t *testing.T) {
	slice := []string{"t🐳st", "in", "slice"}

	test := InSlice(slice, "t🐳st")
	if !test {
		t.Fatalf("Expected string t🐳st to be in slice")
	}
	test = InSlice(slice, "SLICE")
	if !test {
		t.Fatalf("Expected string SLICE to be in slice")
	}
	test = InSlice(slice, "notinslice")
	if test {
		t.Fatalf("Expected string notinslice not to be in slice")
	}
}

func TestShellQuoteArgumentsEmpty(t *testing.T) {
	actual := ShellQuoteArguments([]string{})
	expected := ""
	if actual != expected {
		t.Fatalf("Expected an empty string")
	}
}

func TestShellQuoteArguments(t *testing.T) {
	simpleString := "simpleString"
	complexString := "This is a 'more' complex $string with some special char *"
	actual := ShellQuoteArguments([]string{simpleString, complexString})
	expected := "simpleString 'This is a '\\''more'\\'' complex $string with some special char *'"
	if actual != expected {
		t.Fatalf("Expected \"%v\", got \"%v\"", expected, actual)
	}
}
