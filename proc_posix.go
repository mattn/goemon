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
		if sig == os.Kill {
			return g.cmd.Process.Kill()
		}
		if err := g.cmd.Process.Signal(sig); err != nil {
			g.Logger.Println(err)
			return g.cmd.Process.Kill()
		}

		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if g.cmd.ProcessState != nil && g.cmd.ProcessState.Exited() {
				return nil
			}
			time.Sleep(100)
		}
		return kill(g.cmd.Process)
	}
	return nil
}
