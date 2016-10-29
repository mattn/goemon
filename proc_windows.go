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

func (g *goemon) terminate() error {
	if g.cmd != nil && g.cmd.Process != nil {
		if err := interrupt(g.cmd.Process); err != nil {
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
	}
	return nil
}

func interrupt(p *os.Process) error {
	procSetConsoleCtrlHandler.Call(0, 1)
	defer procSetConsoleCtrlHandler.Call(0, 0)
	r1, _, err := procGenerateConsoleCtrlEvent.Call(syscall.CTRL_BREAK_EVENT, uintptr(p.Pid))
	if r1 == 0 {
		return err
	}
	r1, _, err = procGenerateConsoleCtrlEvent.Call(syscall.CTRL_C_EVENT, uintptr(p.Pid))
	if r1 == 0 {
		return err
	}
	return nil
}
