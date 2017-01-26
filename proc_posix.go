// +build !windows

package goemon

import (
	"os"
	"os/exec"
	"time"
)

func (g *goemon) spawn() error {
	g.cmd = exec.Command(g.Args[0], g.Args[1:]...)
	g.cmd.Stdout = os.Stdout
	g.cmd.Stderr = os.Stderr
	return g.cmd.Run()
}

func (g *goemon) terminate() error {
	if g.cmd != nil && g.cmd.Process != nil {
		if err := g.cmd.Process.Signal(os.Interrupt); err != nil {
			g.Logger.Println(err)
		} else {
			cd := 5
			for cd > 0 {
				if g.cmd.ProcessState != nil {
					break
				}
				time.Sleep(time.Second)
				cd--
			}
		}
		if g.cmd.ProcessState == nil {
			g.cmd.Process.Kill()
		}
	}
	return nil
}
