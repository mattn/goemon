// +build !windows

package goemon

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestTerminateProcessGroup(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "goemon")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	pidfile := filepath.Join(dir, "pid")

	g := New()
	g.Args = []string{"sh", "-c", "sh -c 'echo $$ > " + pidfile + "; exec sleep 30'"}

	done := make(chan error, 1)
	go func() {
		done <- g.spawn()
	}()

	pid := 0
	for i := 0; i < 50; i++ {
		if b, err := ioutil.ReadFile(pidfile); err == nil && len(b) > 0 {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("could not get grandchild pid")
	}

	err = g.terminate(os.Interrupt)
	if err != nil {
		t.Fatal("Should be succeeded", err)
	}
	<-done

	for i := 0; i < 50; i++ {
		if syscall.Kill(pid, 0) != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	syscall.Kill(pid, syscall.SIGKILL)
	t.Fatal("grandchild process should be terminated")
}
