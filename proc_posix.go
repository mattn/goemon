// +build !windows

package goemon

import (
	"os"
	"os/exec"
	"time"
)

func (g *Goemon) spawn() error {
	g.cmd = exec.Command(g.Args[0], g.Args[1:]...)
	g.cmd.Stdout = os.Stdout
	g.cmd.Stderr = os.Stderr
	return g.cmd.Run()
}

func (g *Goemon) terminate(sig os.Signal) error {
	if sig == nil {
		sig = os.Interrupt
	}
	cmd := g.cmd
	if cmd != nil && cmd.Process != nil {
		if sig == os.Kill {
			return cmd.Process.Kill()
		}
		if err := cmd.Process.Signal(sig); err != nil {
			g.Logger.Println(err)
			return cmd.Process.Kill()
		}

		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
		return cmd.Process.Kill()
	}
	return nil
}
