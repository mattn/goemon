// +build !windows

package goemon

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

func (g *Goemon) spawn() error {
	g.cmd = exec.Command(g.Args[0], g.Args[1:]...)
	g.cmd.Stdout = os.Stdout
	g.cmd.Stderr = os.Stderr
	// Run the command in its own process group so that terminate can
	// signal the command and all of its descendants at once.
	g.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return g.cmd.Run()
}

func signalGroup(p *os.Process, sig os.Signal) error {
	s, ok := sig.(syscall.Signal)
	if !ok {
		return p.Signal(sig)
	}
	err := syscall.Kill(-p.Pid, s)
	if err == nil || err == syscall.ESRCH {
		return nil
	}
	return p.Signal(sig)
}

func killGroup(p *os.Process) error {
	err := syscall.Kill(-p.Pid, syscall.SIGKILL)
	if err == nil || err == syscall.ESRCH {
		return nil
	}
	return p.Kill()
}

func (g *Goemon) terminate(sig os.Signal) error {
	if sig == nil {
		sig = os.Interrupt
	}
	cmd := g.cmd
	if cmd != nil && cmd.Process != nil {
		if sig == os.Kill {
			return killGroup(cmd.Process)
		}
		if err := signalGroup(cmd.Process, sig); err != nil {
			g.Logger.Println(err)
			return killGroup(cmd.Process)
		}

		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if syscall.Kill(-cmd.Process.Pid, 0) == syscall.ESRCH {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
		return killGroup(cmd.Process)
	}
	return nil
}
