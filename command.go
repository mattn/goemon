package goemon

import (
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/omeid/jsmin"
	"github.com/omeid/livereload"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
)

func (g *goemon) internalCommand(command, file string) bool {
	ss := commandRe.FindStringSubmatch(command)
	switch ss[1] {
	case ":livereload":
		for _, s := range ss[2:] {
			g.Logger.Println("reloading", s)
			g.lrs.Reload(s, true)
		}
		return true
	case ":sleep":
		for _, s := range ss[2:] {
			si, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				g.Logger.Println("failed to parse argument for :sleep command:", err)
				return false
			}
			g.Logger.Println("sleeping", s+"ms")
			time.Sleep(time.Duration(si) * time.Microsecond)
		}
		return true
	case ":fizzbuzz":
		for _, s := range ss[2:] {
			si, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				g.Logger.Println("failed to parse argument for :fizzbuzz command:", err)
				return false
			}
			for i := int64(1); i <= si; i++ {
				switch {
				case i%15 == 0:
					g.Logger.Println("FizzBuzz")
				case i%3 == 0:
					g.Logger.Println("Fizz")
				case i%5 == 0:
					g.Logger.Println("Buzz")
				default:
					g.Logger.Println(i)
				}
			}
		}
		return true
	case ":minify":
		return g.minify(file)
	case ":restart":
		g.terminate()
		return true
	case ":event":
		for _, s := range ss[2:] {
			g.Logger.Println("fire", s)
			g.task(fsnotify.Event{Name: s, Op: fsnotify.Write})
		}
	}
	return false
}

func (g *goemon) externalCommand(command, file string) bool {
	var cmd *exec.Cmd
	command = os.Expand(command, func(s string) string {
		switch s {
		case "GOEMON_TARGET_FILE":
			return file
		case "GOEMON_TARGET_BASE":
			return filepath.Base(file)
		case "GOEMON_TARGET_DIR":
			return filepath.ToSlash(filepath.Dir(file))
		case "GOEMON_TARGET_EXT":
			return filepath.Ext(file)
		case "GOEMON_TARGET_NAME":
			fn := filepath.Base(file)
			ext := filepath.Ext(file)
			return fn[:len(fn)-len(ext)]
		}
		return os.Getenv(s)
	})
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	g.Logger.Println("executing", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		g.Logger.Println(err)
		return false
	}
	return true
}

func (g *goemon) minify(name string) bool {
	if strings.HasSuffix(filepath.Base(name), ".min.") {
		return true // ignore
	}
	ext := filepath.Ext(name)
	if ext == "" {
		return true // ignore
	}
	in, err := os.Open(name)
	if err != nil {
		g.Logger.Println(err)
		return false
	}
	defer in.Close()

	switch ext {
	case ".js":

		buf, err := jsmin.Minify(in)
		if err != nil {
			g.Logger.Println(err)
			return false
		}
		err = ioutil.WriteFile(name[:len(name)-len(ext)]+".min.js", buf.Bytes(), 0644)
		if err != nil {
			g.Logger.Println(err)
			return false
		}
		return true
	case ".css":
		out, err := os.Create(name[:len(name)-len(ext)] + ".min.css")
		if err != nil {
			g.Logger.Println(err)
			return false
		}
		m := minify.New()
		m.AddFunc("text/css", css.Minify)
		if err := m.Minify("text/css", out, in); err != nil {
			g.Logger.Println(err)
			return false
		}
		return true
	}
	return false
}

func (g *goemon) livereload() error {
	g.lrs = livereload.New("goemon")
	defer g.lrs.Close()
	addr := g.conf.LiveReload
	if addr == "" {
		addr = os.Getenv("GOEMON_LIVERELOAD_ADDR")
	}
	if addr == "" {
		addr = ":35730"
	}
	var err error
	g.lrc, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer g.lrc.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("/livereload.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, err := w.Write([]byte(liveReloadScript))
		if err != nil {
			g.lrc.Close()
		}
	})
	mux.Handle("/livereload", g.lrs)
	return http.Serve(g.lrc, mux)
}
