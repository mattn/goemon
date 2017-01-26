package main

//go:generate go-bindata -prefix assets assets

import (
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/mattn/goemon"
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
				fmt.Print(string(MustAsset("web.yml")))
			} else if os.Args[2] == "?" {
				keys := AssetNames()
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Println(k[:len(k)-4])
				}
			} else if t, err := Asset(os.Args[2] + ".yml"); err == nil {
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
