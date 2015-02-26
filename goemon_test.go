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
	if g.jsmin(f) {
		t.Fatal("Should not be succeeded")
	}
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
	if !g.jsmin(f) {
		t.Fatal("Should be succeeded")
	}
	_, err = os.Stat(filepath.Join(dir, "foo.min.js"))
	if err != nil {
		t.Fatal(t)
	}
}

func TestSpawn(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "goemon")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	g := New()
	g.Args = []string{"go", "version"}

	err = g.terminate()
	if err != nil {
		t.Fatal("Should be succeeded", err)
	}
	err = g.spawn()
	if err != nil {
		t.Fatal("Should be succeeded", err)
	}
}

func TestLoad(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "goemon")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tmp, err := ioutil.TempFile(dir, "goemon")
	if err != nil {
		t.Fatal(err)
	}

	ioutil.WriteFile(tmp.Name(), []byte(``), 0644)

	g := New()
	g.File = tmp.Name()
	err = g.load()
	if err != nil {
		t.Fatal("Should be succeeded", err)
	}

	ioutil.WriteFile(tmp.Name(), []byte(`asdfasdf`), 0644)

	err = g.load()
	if err == nil {
		t.Fatal("Should not be succeeded")
	}

	ioutil.WriteFile(tmp.Name(), []byte(`
tasks:
- match: './assets/*.js'
  commands:
`), 0644)

	err = g.load()
	if err != nil {
		t.Fatal("Should be succeeded")
	}

	if len(g.conf.Tasks) != 1 {
		t.Fatal("Should have a task at least")
	}
}

func TestMatch(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "goemon")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tmp, err := ioutil.TempFile(dir, "goemon")
	if err != nil {
		t.Fatal(err)
	}

	ioutil.WriteFile(tmp.Name(), []byte(`
tasks:
- match: './assets/*.js'
  commands:
`), 0644)

	g := New()
	g.File = tmp.Name()
	err = g.load()
	if err != nil {
		t.Fatal("Should be succeeded", err)
	}

	tests := []struct {
		file   string
		result bool
	}{
		{"foo", false},
		{"assets", false},
		{"assets/js", false},
		{"assets/.js", false},
		{"assets/a.js", true},
	}

	for _, test := range tests {
		file, _ := filepath.Abs(test.file)
		file = filepath.ToSlash(file)
		if g.conf.Tasks[0].match(file) {
			if !test.result {
				t.Fatal("Should not match:", test.file)
			}
		} else {
			if test.result {
				t.Fatal("Should be match:", test.file)
			}
		}
	}

	ioutil.WriteFile(tmp.Name(), []byte(`
tasks:
- match: './assets/*/*.js'
  commands:
`), 0644)

	err = g.load()
	if err != nil {
		t.Fatal("Should be succeeded", err)
	}

	tests = []struct {
		file   string
		result bool
	}{
		{"assets/a.js", false},
		{"assets/a/.js", false},
		{"assets/a/foooo.js", true},
		{"assets/a/foo/bar.js", false},
	}

	for _, test := range tests {
		file, _ := filepath.Abs(test.file)
		file = filepath.ToSlash(file)
		if g.conf.Tasks[0].match(file) {
			if !test.result {
				t.Fatal("Should not match:", test.file)
			}
		} else {
			if test.result {
				t.Fatal("Should be match:", test.file)
			}
		}
	}

	ioutil.WriteFile(tmp.Name(), []byte(`
tasks:
- match: './assets/*/**.js'
  commands:
`), 0644)

	err = g.load()
	if err != nil {
		t.Fatal("Should be succeeded", err)
	}

	tests = []struct {
		file   string
		result bool
	}{
		{"assets/a/foooo.js", true},
		{"assets/a/foo/bar.js", true},
		{"assets/a/foo/baz/bar.js", true},
	}

	for _, test := range tests {
		file, _ := filepath.Abs(test.file)
		file = filepath.ToSlash(file)
		if g.conf.Tasks[0].match(file) {
			if !test.result {
				t.Fatal("Should not match:", test.file)
			}
		} else {
			if test.result {
				t.Fatal("Should be match:", test.file)
			}
		}
	}
}
