package goemon

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCompilePattern(t *testing.T) {

	tests := []struct {
		re   string
		path string
	}{
		{`/path/**/*.txt`, `/path/to/file.txt`},
	}
	for _, test := range tests {
		if runtime.GOOS == "windows" {
			if p, err := filepath.Abs(test.path); err == nil {
				test.path = p
			}
		}
		re, err := compilePattern(test.re)
		if err != nil {
			t.Fatal(err)
		}
		if !re.MatchString(test.path) {
			t.Fatalf("%v should match as %v: %v", test.re, test.path, re.String())
		}
	}
}

func TestJsmin(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "goemon")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	g := New()
	f := filepath.Join(dir, "foo.js")
	g.jsmin(f)
	_, err = os.Stat(filepath.Join(dir, "foo.min.js"))
	if err == nil {
		t.Fatalf("Should be fail for non-exists file: %v", err)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("Should be fail for non-exists file: %v", err)
	}
	err = ioutil.WriteFile(f, []byte(``), 0644)
	if err != nil {
		t.Fatal(err)
	}
	g.jsmin(f)
	_, err = os.Stat(filepath.Join(dir, "foo.min.js"))
	if err != nil {
		t.Fatal(t)
	}
	// TODO: more tests
}
