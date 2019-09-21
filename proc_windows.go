// +build windows

package goemon

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

var (
	libkernel32                  = syscall.MustLoadDLL("kernel32")
	procSetConsoleCtrlHandler    = libkernel32.MustFindProc("SetConsoleCtrlHandler")
	procGenerateConsoleCtrlEvent = libkernel32.MustFindProc("GenerateConsoleCtrlEvent")
)

func (g *goemon) spawn() error {
	g.cmd = exec.Command(g.Args[0], g.Args[1:]...)
	g.cmd.Stdout = os.Stdout
	g.cmd.Stderr = os.Stderr
	g.cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_UNICODE_ENVIRONMENT | 0x00000200,
	}
	return g.cmd.Run()
}

func (g *goemon) terminate(sig os.Signal) error {
	if g.cmd != nil && g.cmd.Process != nil {
		if err := interrupt(g.cmd.Process, sig); err != nil {
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

func interrupt(p *os.Process, sig os.Signal) error {
	procSetConsoleCtrlHandler.Call(0, 1)
	defer procSetConsoleCtrlHandler.Call(0, 0)
	r1, _, err := procGenerateConsoleCtrlEvent.Call(syscall.CTRL_C_EVENT, uintptr(p.Pid))
	if r1 == 0 {
		return err
	}
	return nil
}
