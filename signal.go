// +build !windows

package goemon

import (
	"os"
)

func interrupt(p *os.Process) error {
	return p.Signal(os.Interrupt)
}
