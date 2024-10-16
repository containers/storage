package fileutils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const windows = "windows"

// CopyFile with invalid src
func TestCopyFileWithInvalidSrc(t *testing.T) {
	tempFolder := t.TempDir()
	bytes, err := CopyFile("/invalid/file/path", path.Join(tempFolder, "dest"))
	if err == nil {
		t.Fatal("Should have fail to copy an invalid src file")
	}
	if bytes != 0 {
		t.Fatal("Should have written 0 bytes")
	}
}

// CopyFile with invalid dest
func TestCopyFileWithInvalidDest(t *testing.T) {
	tempFolder := t.TempDir()
	src := path.Join(tempFolder, "file")
	err := os.WriteFile(src, []byte("content"), 0o740)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := CopyFile(src, path.Join(tempFolder, "/invalid/dest/path"))
	if err == nil {
		t.Fatal("Should have fail to copy an invalid src file")
	}
	if bytes != 0 {
		t.Fatal("Should have written 0 bytes")
	}
}

// CopyFile with same src and dest
func TestCopyFileWithSameSrcAndDest(t *testing.T) {
	tempFolder := t.TempDir()
	file := path.Join(tempFolder, "file")
	err := os.WriteFile(file, []byte("content"), 0o740)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := CopyFile(file, file)
	if err != nil {
		t.Fatal(err)
	}
	if bytes != 0 {
		t.Fatal("Should have written 0 bytes as it is the same file.")
	}
}

// CopyFile with same src and dest but path is different and not clean
func TestCopyFileWithSameSrcAndDestWithPathNameDifferent(t *testing.T) {
	tempFolder := t.TempDir()
	testFolder := path.Join(tempFolder, "test")
	err := os.MkdirAll(testFolder, 0o740)
	if err != nil {
		t.Fatal(err)
	}
	file := path.Join(testFolder, "file")
	sameFile := testFolder + "/../test/file"
	err = os.WriteFile(file, []byte("content"), 0o740)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := CopyFile(file, sameFile)
	if err != nil {
		t.Fatal(err)
	}
	if bytes != 0 {
		t.Fatal("Should have written 0 bytes as it is the same file.")
	}
}

func TestCopyFile(t *testing.T) {
	tempFolder := t.TempDir()
	src := path.Join(tempFolder, "src")
	dest := path.Join(tempFolder, "dest")
	err := os.WriteFile(src, []byte("content"), 0o777)
	require.NoError(t, err)

	err = os.WriteFile(dest, []byte("destContent"), 0o777)
	require.NoError(t, err)
	bytes, err := CopyFile(src, dest)
	if err != nil {
		t.Fatal(err)
	}
	if bytes != 7 {
		t.Fatalf("Should have written %d bytes but wrote %d", 7, bytes)
	}
	actual, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != "content" {
		t.Fatalf("Dest content was '%s', expected '%s'", string(actual), "content")
	}
}

// Reading a symlink to a directory must return the directory
func TestReadSymlinkedDirectoryExistingDirectory(t *testing.T) {
	// TODO Windows: Port this test
	if runtime.GOOS == windows {
		t.Skip("Needs porting to Windows")
	}
	var err error
	if err = os.Mkdir("/tmp/testReadSymlinkToExistingDirectory", 0o777); err != nil {
		t.Errorf("failed to create directory: %s", err)
	}

	if err = os.Symlink("/tmp/testReadSymlinkToExistingDirectory", "/tmp/dirLinkTest"); err != nil {
		t.Errorf("failed to create symlink: %s", err)
	}

	var path string
	if path, err = ReadSymlinkedDirectory("/tmp/dirLinkTest"); err != nil {
		t.Fatalf("failed to read symlink to directory: %s", err)
	}

	if path != "/tmp/testReadSymlinkToExistingDirectory" {
		t.Fatalf("symlink returned unexpected directory: %s", path)
	}

	if err = os.Remove("/tmp/testReadSymlinkToExistingDirectory"); err != nil {
		t.Errorf("failed to remove temporary directory: %s", err)
	}

	if err = os.Remove("/tmp/dirLinkTest"); err != nil {
		t.Errorf("failed to remove symlink: %s", err)
	}
}

// Reading a non-existing symlink must fail
func TestReadSymlinkedDirectoryNonExistingSymlink(t *testing.T) {
	var path string
	var err error
	if path, err = ReadSymlinkedDirectory("/tmp/test/foo/Non/ExistingPath"); err == nil {
		t.Fatalf("error expected for non-existing symlink")
	}

	if path != "" {
		t.Fatalf("expected empty path, but '%s' was returned", path)
	}
}

// Reading a symlink to a file must fail
func TestReadSymlinkedDirectoryToFile(t *testing.T) {
	// TODO Windows: Port this test
	if runtime.GOOS == windows {
		t.Skip("Needs porting to Windows")
	}
	var err error
	var file *os.File

	if file, err = os.Create("/tmp/testReadSymlinkToFile"); err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	file.Close()

	if err = os.Symlink("/tmp/testReadSymlinkToFile", "/tmp/fileLinkTest"); err != nil {
		t.Errorf("failed to create symlink: %s", err)
	}

	var path string
	if path, err = ReadSymlinkedDirectory("/tmp/fileLinkTest"); err == nil {
		t.Fatalf("ReadSymlinkedDirectory on a symlink to a file should've failed")
	}

	if path != "" {
		t.Fatalf("path should've been empty: %s", path)
	}

	if err = os.Remove("/tmp/testReadSymlinkToFile"); err != nil {
		t.Errorf("failed to remove file: %s", err)
	}

	if err = os.Remove("/tmp/fileLinkTest"); err != nil {
		t.Errorf("failed to remove symlink: %s", err)
	}
}

func TestWildcardMatches(t *testing.T) {
	match, _ := Matches("fileutils.go", []string{"*"})
	if !match {
		t.Errorf("failed to get a wildcard match, got %v", match)
	}
}

// A simple pattern match should return true.
func TestPatternMatches(t *testing.T) {
	match, _ := Matches("fileutils.go", []string{"*.go"})
	if !match {
		t.Errorf("failed to get a match, got %v", match)
	}
}

// An exclusion followed by an inclusion should return false.
func TestExclusionPatternMatchesPatternBefore(t *testing.T) {
	match, _ := Matches("fileutils.go", []string{"!fileutils.go", "*.go"})
	if !match {
		t.Errorf("failed to get false match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderExclusions(t *testing.T) {
	match, _ := Matches("docs/README.md", []string{"docs", "!docs/README.md"})
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderWithSlashExclusions(t *testing.T) {
	match, _ := Matches("docs/README.md", []string{"docs/", "!docs/README.md"})
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderWildcardExclusions(t *testing.T) {
	match, _ := Matches("docs/README.md", []string{"docs/*", "!docs/README.md"})
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A pattern followed by an exclusion should return false.
func TestExclusionPatternMatchesPatternAfter(t *testing.T) {
	match, _ := Matches("fileutils.go", []string{"*.go", "!fileutils.go"})
	if match {
		t.Errorf("failed to get false match on exclusion pattern, got %v", match)
	}
}

// A filename evaluating to . should return false.
func TestExclusionPatternMatchesWholeDirectory(t *testing.T) {
	match, _ := Matches(".", []string{"*.go"})
	if match {
		t.Errorf("failed to get false match on ., got %v", match)
	}
}

// A single ! pattern should return an error.
func TestSingleExclamationError(t *testing.T) {
	_, err := Matches("fileutils.go", []string{"!"})
	if err == nil {
		t.Errorf("failed to get an error for a single exclamation point, got %v", err)
	}
}

// Matches with no patterns
func TestMatchesWithNoPatterns(t *testing.T) {
	matches, err := Matches("/any/path/there", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if matches {
		t.Fatalf("Should not have match anything")
	}
}

// Matches with malformed patterns
func TestMatchesWithMalformedPatterns(t *testing.T) {
	matches, err := Matches("/any/path/there", []string{"["})
	if err == nil {
		t.Fatal("Should have failed because of a malformed syntax in the pattern")
	}
	if matches {
		t.Fatalf("Should not have match anything")
	}
}

type matchesTestCase struct {
	pattern string
	text    string
	fail    bool
	match   bool
}

func TestMatches(t *testing.T) {
	tests := []matchesTestCase{
		{"**", "file", false, true},
		{"**", "file/", false, true},
		{"**/", "file", false, true}, // weird one
		{"**/", "file/", false, true},
		{"**", "/", false, true},
		{"**/", "/", false, true},
		{"**", "dir/file", false, true},
		{"**/", "dir/file", false, true},
		{"**", "dir/file/", false, true},
		{"**/", "dir/file/", false, true},
		{"**/**", "dir/file", false, true},
		{"**/**", "dir/file/", false, true},
		{"dir/**", "dir/file", false, true},
		{"dir/**", "dir/file/", false, true},
		{"dir/**", "dir/dir2/file", false, true},
		{"dir/**", "dir/dir2/file/", false, true},
		{"**/dir", "dir/", false, true},
		{"**/dir", "dir/file", false, true},
		{"**/dir", "dir/dir2/file", false, true},
		{"**/dir", "dir1/dir/", false, true},
		{"**/dir", "dir1/dir/file", false, true},
		{"**/dir", "dir1/dir/dir2/file", false, true},
		{"**/dir", "dir1/dir2/dir/", false, true},
		{"**/dir", "dir1/dir2/dir/file", false, true},
		{"**/dir", "dir1/dir2/dir/dir3/file", false, true},
		{"**/dir2/*", "dir/dir2/file", false, true},
		{"**/dir2/*", "dir/dir2/file/", false, true},
		{"**/dir2/**", "dir/dir2/dir3/file", false, true},
		{"**/dir2/**", "dir/dir2/dir3/file/", false, true},
		{"**file", "file", false, true},
		{"**file", "dir/file", false, true},
		{"**/file", "dir/file", false, true},
		{"**file", "dir/dir/file", false, true},
		{"**/file", "dir/dir/file", false, true},
		{"**/file*", "dir/dir/file", false, true},
		{"**/file*", "dir/dir/file.txt", false, true},
		{"**/file*txt", "dir/dir/file.txt", false, true},
		{"**/file*.txt", "dir/dir/file.txt", false, true},
		{"**/file*.txt*", "dir/dir/file.txt", false, true},
		{"**/**/*.txt", "dir/dir/file.txt", false, true},
		{"**/**/*.txt2", "dir/dir/file.txt", false, false},
		{"**/*.txt", "file.txt", false, true},
		{"**/**/*.txt", "file.txt", false, true},
		{"a**/*.txt", "a/file.txt", false, true},
		{"a**/*.txt", "a/dir/file.txt", false, true},
		{"a**/*.txt", "a/dir/dir/file.txt", false, true},
		{"a/*.txt", "a/dir/file.txt", false, false},
		{"a/*.txt", "a/file.txt", false, true},
		{"a/*.txt**", "a/file.txt", false, true},
		{"a[b-d]e", "ae", false, false},
		{"a[b-d]e", "ace", false, true},
		{"a[b-d]e", "aae", false, false},
		{"a[^b-d]e", "aze", false, true},
		{".*", ".foo", false, true},
		{".*", "foo", false, false},
		{"abc.def", "abcdef", false, false},
		{"abc.def", "abc.def", false, true},
		{"abc.def", "abcZdef", false, false},
		{"abc?def", "abcZdef", false, true},
		{"abc?def", "abcdef", false, false},
		{"a\\\\", "a\\", false, true},
		{"**/foo/bar", "foo/bar", false, true},
		{"**/foo/bar", "dir/foo/bar", false, true},
		{"**/foo/bar", "dir/dir2/foo/bar", false, true},
		{"abc/**", "abc", false, false},
		{"abc/**", "abc/def", false, true},
		{"abc/**", "abc/def/ghi", false, true},
		{"**/.foo", ".foo", false, true},
		{"**/.foo", "bar.foo", false, false},
	}

	if runtime.GOOS != windows {
		tests = append(tests, []matchesTestCase{
			{"a\\*b", "a*b", false, true},
			{"a\\", "a", true, false},
			{"a\\", "a\\", true, false},
			{"a\\", "a$", true, false},
		}...)
	}

	_, err := filepath.Match("[", "")
	badPatternsCaughtEarly := err == filepath.ErrBadPattern

	for _, test := range tests {
		desc := fmt.Sprintf("pattern=%q text=%q", test.pattern, test.text)
		t.Run(desc, func(t *testing.T) {
			pm, err := NewPatternMatcher([]string{test.pattern})
			if test.fail && badPatternsCaughtEarly {
				assert.Equal(t, err, filepath.ErrBadPattern) // pm is nil, we're done
			} else {
				require.NoError(t, err, desc)
				res, err := pm.MatchesResult(test.text)
				if test.fail {
					assert.Equal(t, err, filepath.ErrBadPattern)
				} else {
					assert.Nil(t, err)
					assert.Equal(t, test.match, res.IsMatched(), desc)
				}
			}
		})
	}
}

func TestCleanPatterns(t *testing.T) {
	patterns := []string{"docs", "config"}
	pm, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("invalid pattern %v", patterns)
	}
	cleaned := pm.Patterns()
	if len(cleaned) != 2 {
		t.Errorf("expected 2 element slice, got %v", len(cleaned))
	}
}

func TestCleanPatternsStripEmptyPatterns(t *testing.T) {
	patterns := []string{"docs", "config", ""}
	pm, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("invalid pattern %v", patterns)
	}
	cleaned := pm.Patterns()
	if len(cleaned) != 2 {
		t.Errorf("expected 2 element slice, got %v", len(cleaned))
	}
}

func TestCleanPatternsExceptionFlag(t *testing.T) {
	patterns := []string{"docs", "!docs/README.md"}
	pm, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("invalid pattern %v", patterns)
	}
	if !pm.Exclusions() {
		t.Errorf("expected exceptions to be true, got %v", pm.Exclusions())
	}
}

func TestCleanPatternsLeadingSpaceTrimmed(t *testing.T) {
	patterns := []string{"docs", "  !docs/README.md"}
	pm, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("invalid pattern %v", patterns)
	}
	if !pm.Exclusions() {
		t.Errorf("expected exceptions to be true, got %v", pm.Exclusions())
	}
}

func TestCleanPatternsTrailingSpaceTrimmed(t *testing.T) {
	patterns := []string{"docs", "!docs/README.md  "}
	pm, err := NewPatternMatcher(patterns)
	if err != nil {
		t.Fatalf("invalid pattern %v", patterns)
	}
	if !pm.Exclusions() {
		t.Errorf("expected exceptions to be true, got %v", pm.Exclusions())
	}
}

func TestCleanPatternsErrorSingleException(t *testing.T) {
	patterns := []string{"!"}
	_, err := NewPatternMatcher(patterns)
	if err == nil {
		t.Errorf("expected error on single exclamation point, got %v", err)
	}
}

func TestCreateIfNotExistsDir(t *testing.T) {
	tempFolder := t.TempDir()

	folderToCreate := filepath.Join(tempFolder, "tocreate")

	if err := CreateIfNotExists(folderToCreate, true); err != nil {
		t.Fatal(err)
	}
	fileinfo, err := os.Stat(folderToCreate)
	if err != nil {
		t.Fatalf("Should have create a folder, got %v", err)
	}

	if !fileinfo.IsDir() {
		t.Fatalf("Should have been a dir, seems it's not")
	}
}

func TestCreateIfNotExistsFile(t *testing.T) {
	tempFolder := t.TempDir()

	fileToCreate := filepath.Join(tempFolder, "file/to/create")

	if err := CreateIfNotExists(fileToCreate, false); err != nil {
		t.Fatal(err)
	}
	fileinfo, err := os.Stat(fileToCreate)
	if err != nil {
		t.Fatalf("Should have create a file, got %v", err)
	}

	if fileinfo.IsDir() {
		t.Fatalf("Should have been a file, seems it's not")
	}
}

// These matchTests are stolen from go's filepath Match tests.
type matchTest struct {
	pattern, s string
	match      bool
	err        error
}

var matchTests = []matchTest{
	{"abc", "abc", true, nil},
	{"*", "abc", true, nil},
	{"*c", "abc", true, nil},
	{"a*", "a", true, nil},
	{"a*", "abc", true, nil},
	{"a*", "ab/c", true, nil},
	{"a*/b", "abc/b", true, nil},
	{"a*/b", "a/c/b", false, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil},
	{"a*b?c*x", "abxbbxdbxebxczzx", true, nil},
	{"a*b?c*x", "abxbbxdbxebxczzy", false, nil},
	{"ab[c]", "abc", true, nil},
	{"ab[b-d]", "abc", true, nil},
	{"ab[e-g]", "abc", false, nil},
	{"ab[^c]", "abc", false, nil},
	{"ab[^b-d]", "abc", false, nil},
	{"ab[^e-g]", "abc", true, nil},
	{"a\\*b", "a*b", true, nil},
	{"a\\*b", "ab", false, nil},
	{"a?b", "a☺b", true, nil},
	{"a[^a]b", "a☺b", true, nil},
	{"a???b", "a☺b", false, nil},
	{"a[^a][^a][^a]b", "a☺b", false, nil},
	{"[a-ζ]*", "α", true, nil},
	{"*[a-ζ]", "A", false, nil},
	{"a?b", "a/b", false, nil},
	{"a*b", "a/b", false, nil},
	{"[\\]a]", "]", true, nil},
	{"[\\-]", "-", true, nil},
	{"[x\\-]", "x", true, nil},
	{"[x\\-]", "-", true, nil},
	{"[x\\-]", "z", false, nil},
	{"[\\-x]", "x", true, nil},
	{"[\\-x]", "-", true, nil},
	{"[\\-x]", "a", false, nil},
	{"[]a]", "]", false, filepath.ErrBadPattern},
	{"[-]", "-", false, filepath.ErrBadPattern},
	{"[x-]", "x", false, filepath.ErrBadPattern},
	{"[x-]", "-", false, filepath.ErrBadPattern},
	{"[x-]", "z", false, filepath.ErrBadPattern},
	{"[-x]", "x", false, filepath.ErrBadPattern},
	{"[-x]", "-", false, filepath.ErrBadPattern},
	{"[-x]", "a", false, filepath.ErrBadPattern},
	{"\\", "a", false, filepath.ErrBadPattern},
	{"[a-b-c]", "a", false, filepath.ErrBadPattern},
	{"[", "a", false, filepath.ErrBadPattern},
	{"[^", "a", false, filepath.ErrBadPattern},
	{"[^bc", "a", false, filepath.ErrBadPattern},
	{"a[", "a", false, filepath.ErrBadPattern}, // was nil but IMO its wrong
	{"a[", "ab", false, filepath.ErrBadPattern},
	{"*x", "xxx", true, nil},
}

func errp(e error) string {
	if e == nil {
		return "<nil>"
	}
	return e.Error()
}

// TestMatch test's our version of filepath.Match, called regexpMatch.
func TestMatch(t *testing.T) {
	for _, tt := range matchTests {
		pattern := tt.pattern
		s := tt.s
		if runtime.GOOS == windows {
			if strings.Contains(pattern, "\\") {
				// no escape allowed on windows.
				continue
			}
			pattern = filepath.Clean(pattern)
			s = filepath.Clean(s)
		}
		ok, err := Matches(s, []string{pattern})
		if ok != tt.match || err != tt.err {
			t.Fatalf("Match(%#q, %#q) = %v, %q want %v, %q", pattern, s, ok, errp(err), tt.match, errp(tt.err))
		}
	}
}

func TestMatchesAmount(t *testing.T) {
	testData := []struct {
		patterns   []string
		input      string
		matches    uint
		excludes   uint
		isMatch    bool
		canSkipDir bool
	}{
		{[]string{"1", "2", "3"}, "2", 1, 0, true, true},
		{[]string{"!1", "1"}, "1", 1, 1, true, true},
		{[]string{"2", "!2", "!2"}, "2", 1, 2, false, false},
		{[]string{"1", "2", "2"}, "2", 2, 0, true, true},
		{[]string{"1", "2", "2", "2"}, "2", 3, 0, true, true},
		{[]string{"/prefix/path", "/prefix/other"}, "/prefix/path", 1, 0, true, true},
		{[]string{"/prefix*", "!/prefix/path"}, "/prefix/match", 1, 0, true, false},
		{[]string{"/prefix*", "!/prefix/path"}, "/prefix/path", 1, 0, true, false},
		{[]string{"/prefix*", "!/prefix/path"}, "prefix/path", 0, 1, false, false},
		{[]string{"/prefix*", "!./prefix/path"}, "prefix/path", 0, 1, false, false},
		{[]string{"/prefix*", "!prefix/path"}, "prefix/path", 0, 1, false, false},
	}

	for _, testCase := range testData {
		pm, err := NewPatternMatcher(testCase.patterns)
		require.NoError(t, err)
		res, err := pm.MatchesResult(testCase.input)
		require.NoError(t, err)
		desc := fmt.Sprintf("pattern=%q input=%q", testCase.patterns, testCase.input)
		assert.Equal(t, testCase.excludes, res.Excludes(), desc)
		assert.Equal(t, testCase.matches, res.Matches(), desc)
		assert.Equal(t, testCase.isMatch, res.IsMatched(), desc)
		assert.Equal(t, testCase.canSkipDir, res.CanSkipDir(), desc)

		isMatch, err := pm.IsMatch(testCase.input)
		require.NoError(t, err)
		assert.Equal(t, testCase.isMatch, isMatch, desc)
	}
}
