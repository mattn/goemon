package goemon

import (
	"bytes"
	"fmt"
	"github.com/omeid/jsmin"
	"github.com/omeid/livereload"
	"gopkg.in/fsnotify.v1"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

var commandRe = regexp.MustCompile(`^\s*(:[a-z]+)(?:\s+(\S+))*$`)

type goemon struct {
	File   string
	Logger *log.Logger
	Args   []string
	lrc    net.Listener
	lrs    *livereload.Server
	fsw    *fsnotify.Watcher
	cmd    *exec.Cmd
	conf   conf
}

type task struct {
	Match    string   `yaml:"match"`
	Commands []string `yaml:"commands"`
	mre      *regexp.Regexp
	ire      *regexp.Regexp
	hit      bool
	mutex    sync.Mutex
}

type conf struct {
	Command    string
	LiveReload string  `yaml:"livereload"`
	Tasks      []*task `yaml:"tasks"`
}

func compilePattern(pattern string) (*regexp.Regexp, error) {
	var buf bytes.Buffer
	buf.WriteString("^")
	if fs, err := filepath.Abs(pattern); err == nil {
		pattern = filepath.ToSlash(fs)
	}
	rs := []rune(pattern)
	for i := 0; i < len(rs); i++ {
		if rs[i] == '/' {
			if runtime.GOOS == "windows" {
				buf.WriteString(`[/\\]`)
			} else {
				buf.WriteRune(rs[i])
			}
		} else if rs[i] == '*' {
			if i < len(rs)-1 && rs[i+1] == '*' {
				buf.WriteString(`.*`)
				i++
			} else {
				buf.WriteString(`[^/]+`)
			}
		} else if rs[i] == '?' {
			buf.WriteString(`\S`)
		} else {
			buf.WriteString(fmt.Sprintf(`[\x%x]`, rs[i]))
		}
	}
	buf.WriteString("$")
	return regexp.Compile(buf.String())
}

func (g *goemon) jsmin(name string) {
	if strings.HasSuffix(name, ".min.js") {
		return
	}
	ext := filepath.Ext(name)
	if ext == "" {
		return
	}
	f, err := os.Open(name)
	if err != nil {
		g.Logger.Println(err)
		return
	}
	defer f.Close()
	buf, err := jsmin.Minify(f)
	if err != nil {
		g.Logger.Println(err)
		return
	}
	err = ioutil.WriteFile(name[:len(name)-len(ext)]+".min.js", buf.Bytes(), 0644)
	if err != nil {
		g.Logger.Println(err)
	}
}

func (g *goemon) terminate() error {
	if err := g.cmd.Process.Signal(os.Interrupt); err != nil {
		g.Logger.Println(err)
	} else {
		cd := 5
		for cd > 0 {
			if g.cmd.ProcessState != nil && g.cmd.ProcessState.Exited() {
				break
			}
			time.Sleep(time.Second)
			cd--
		}
	}
	if g.cmd.ProcessState != nil && g.cmd.ProcessState.Exited() {
		g.cmd.Process.Kill()
	}
	return nil
}

func (g *goemon) restart() error {
	if len(g.Args) == 0 {
		return nil
	}
	g.terminate()
	g.cmd = exec.Command(g.Args[0], g.Args[1:]...)
	g.cmd.Stdout = os.Stdout
	g.cmd.Stderr = os.Stderr
	return g.cmd.Run()
}

func (g *goemon) task(event fsnotify.Event) {
	file := filepath.ToSlash(event.Name)
	for _, t := range g.conf.Tasks {
		if !t.mre.MatchString(file) {
			continue
		}
		t.mutex.Lock()
		if t.hit {
			t.mutex.Unlock()
			continue
		}
		t.hit = true
		t.mutex.Unlock()
		g.Logger.Println(event)
		go func(name string, t *task) {
			for _, command := range t.Commands {
				switch {
				case commandRe.MatchString(command):
					ss := commandRe.FindStringSubmatch(command)
					switch ss[1] {
					case ":livereload":
						for _, s := range ss[2:] {
							g.Logger.Println("reloading", s)
							g.lrs.Reload(s)
						}
					case ":jsmin":
						g.jsmin(name)
					case ":restart":
						g.terminate()
					}
				default:
					var cmd *exec.Cmd
					command = os.Expand(command, func(s string) string {
						if s == "GOEMON_TARGET_FILE" {
							return name
						}
						if s == "GOEMON_TARGET_BASE" {
							return filepath.Base(name)
						}
						if s == "GOEMON_TARGET_DIR" {
							return filepath.Dir(name)
						}
						if s == "GOEMON_TARGET_BASE" {
							return filepath.Base(name)
						}
						if s == "GOEMON_TARGET_EXT" {
							return filepath.Ext(name)
						}
						if s == "GOEMON_TARGET_NAME" {
							fn := filepath.Base(name)
							ext := filepath.Ext(name)
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
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					err := cmd.Run()
					if err != nil {
						g.Logger.Println(err)
					}
				}
			}
			t.mutex.Lock()
			t.hit = false
			t.mutex.Unlock()
		}(event.Name, t)
	}
}

func (g *goemon) watch() error {
	var err error
	g.fsw, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	g.fsw.Add(g.File)

	root, err := filepath.Abs(".")
	if err != nil {
		g.Logger.Println(err)
	}

	dup := map[string]bool{}
	g.fsw.Add(root)
	dup[root] = true

	for _, t := range g.conf.Tasks {
		if t.Match != "" {
			t.mre, err = compilePattern(t.Match)
			if err != nil {
				g.Logger.Println(err)
			}
		}
		err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if info == nil {
				return err
			}
			name, err := filepath.Abs(info.Name())
			if err != nil {
				g.Logger.Println(err)
			}
			if !info.IsDir() {
				return nil
			}
			if _, ok := dup[name]; !ok {
				g.fsw.Add(name)
				dup[name] = true
			}
			return nil
		})
		if err != nil {
			g.Logger.Println(err)
		}
	}

	g.Logger.Println("goemon loaded", g.File)

	for {
		select {
		case event := <-g.fsw.Events:
			if event.Name == g.File {
				return nil
			}
			g.task(event)
		case err := <-g.fsw.Errors:
			if err != nil {
				g.Logger.Println("error:", err)
			}
		}
	}
}

func (g *goemon) livereload() error {
	g.lrs = livereload.New()
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

func New() *goemon {
	return &goemon{
		File:   "goemon.yml",
		Logger: log.New(os.Stderr, "GOEMON ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func NewWithArgs(args []string) *goemon {
	g := New()
	g.Args = args
	return g
}

func (g *goemon) load() error {
	fn, err := filepath.Abs(g.File)
	if err != nil {
		return err
	}
	g.File = fn
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(b, &g.conf)
	if err != nil {
		return err
	}
	if len(g.Args) == 0 && g.conf.Command != "" {
		if runtime.GOOS == "windows" {
			g.Args = []string{"cmd", "/c", g.conf.Command}
		} else {
			g.Args = []string{"sh", "-c", g.conf.Command}
		}
	}
	return nil
}

func (g *goemon) Run() *goemon {
	err := g.load()
	if err != nil {
		g.Logger.Println(err)
	}

	go func() {
		g.Logger.Println("loading", g.File)
		for {
			err := g.watch()
			if err != nil {
				g.Logger.Println(err)
				time.Sleep(time.Second)
			}
			g.Logger.Println("reloading", g.File)
			err = g.load()
			if err != nil {
				g.Logger.Println(err)
				time.Sleep(time.Second)
			}
		}
	}()

	go func() {
		g.Logger.Println("starting livereload")
		for {
			err := g.livereload()
			if err != nil {
				g.Logger.Println(err)
				time.Sleep(time.Second)
			}
			g.Logger.Println("restarting livereload")
		}
	}()

	if len(g.Args) > 0 {
		g.Logger.Println("starting command", g.Args)
		for {
			err := g.restart()
			if err != nil {
				g.Logger.Println(err)
				time.Sleep(time.Second)
			}
			g.Logger.Println("restarting command")
		}
	}
	return g
}

func (g *goemon) Die() {
	if g.lrc != nil {
		g.lrc.Close()
	}
	if g.fsw != nil {
		g.fsw.Close()
	}
	if g.cmd.Process != nil {
		g.terminate()
	}
	g.Logger.Println("goemon terminated")
}

func Run() *goemon {
	return New().Run()
}
