package main

//go:generate go get github.com/rakyll/statik
//go:generate statik

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"

	"github.com/miya-masa/goemon"
	_ "github.com/miya-masa/goemon/cmd/goemon/statik"
	"github.com/rakyll/statik/fs"
)

const (
	name     = "goemon"
	version  = "0.0.1"
	revision = "HEAD"
)

func usage() {
	fmt.Printf("Usage of %s [options] [command] [args...]\n", os.Args[0])
	fmt.Println(" goemon -g [NAME]     : generate default configuration")
	fmt.Println(" goemon -c [FILE] ... : set configuration file")
	fmt.Println("")
	fmt.Println("* Examples:")
	fmt.Println("  Generate default configuration:")
	fmt.Println("    goemon -g > goemon.yml")
	fmt.Println("")
	fmt.Println("  Generate C configuration:")
	fmt.Println("    goemon -g c > goemon.yml")
	fmt.Println("")
	fmt.Println("  List default configurations:")
	fmt.Println("    goemon -g ?")
	fmt.Println("")
	fmt.Println("  Start standalone server:")
	fmt.Println("    goemon --")
	fmt.Println("  Start web server:")
	fmt.Println("    goemon -a :5000")
	os.Exit(1)
}

var hfs http.FileSystem

func init() {
	var err error
	hfs, err = fs.New()
	if err != nil {
		log.Fatal(err)
	}
}

func asset(name string) ([]byte, error) {
	f, err := hfs.Open("/" + name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

func names() []string {
	dir, err := hfs.Open("/")
	if err != nil {
		log.Fatal(err)
	}
	defer dir.Close()
	fss, err := dir.Readdir(-1)
	if err != nil {
		log.Fatal(err)
	}
	var files []string
	for _, fsi := range fss {
		files = append(files, fsi.Name())
	}
	return files
}

func main() {
	file := ""
	args := []string{}
	addr := ""

	switch len(os.Args) {
	case 1:
		usage()
	default:
		switch os.Args[1] {
		case "-h":
			usage()
		case "-g":
			if len(os.Args) == 2 {
				b, _ := asset("web.yml")
				fmt.Print(string(string(b)))
			} else if os.Args[2] == "?" {
				keys := names()
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Println(k[:len(k)-4])
				}
			} else if t, err := asset(os.Args[2] + ".yml"); err == nil {
				fmt.Print(string(t))
			} else {
				usage()
			}
			return
		case "-a":
			if len(os.Args) == 2 {
				usage()
				return
			}
			addr = os.Args[2]
			args = os.Args[3:]
		case "-c":
			if len(os.Args) == 2 {
				usage()
				return
			}
			file = os.Args[2]
			args = os.Args[3:]
		case "--":
			args = os.Args[2:]
		case "-v":
			fmt.Printf("%s %s (rev: %s/%s)\n", name, version, revision, runtime.Version())
			os.Exit(1)
		default:
			args = os.Args[1:]
		}
	}

	g := goemon.NewWithArgs(args)
	if file != "" {
		g.File = file
	}
	g.Run()
	if len(args) == 0 {
		if addr != "" {
			http.Handle("/", http.FileServer(http.Dir(".")))
			http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				g.Logger.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
				http.DefaultServeMux.ServeHTTP(w, r)
			}))
		} else {
			select {}
		}
	}
}
