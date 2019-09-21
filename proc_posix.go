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

func (g *goemon) terminate(sig os.Signal) error {
	if g.cmd != nil && g.cmd.Process != nil {
		if err := g.cmd.Process.Signal(sig); err != nil {
			g.Logger.Println(err)
			return g.cmd.Process.Kill()
		}
		t := time.AfterFunc(5*time.Second, func() {
			g.cmd.Process.Kill()
		})
		defer t.Stop()
		return g.cmd.Wait()
	}
	return nil
}
