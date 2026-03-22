package internal

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

type linuxProcessSlayer struct{}

func sendProcessSignal(pid int, sig os.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	err = proc.Signal(sig) 
	return err
}

func (ps *linuxProcessSlayer) TermProcess(pid int) error {
	err := sendProcessSignal(pid, syscall.SIGTERM)
	fmt.Printf("SIGTERM sent to pid %v\n", pid)

	time.Sleep(500 * time.Millisecond)
	if err := sendProcessSignal(pid, syscall.Signal(0)); err == nil {
		// process still exists
		return fmt.Errorf("pid %d still running after 500ms, retry with --kill/-k to force\n", pid)
	}
	return err
}

func (ps *linuxProcessSlayer) KillProcess(pid int) error {
	err := sendProcessSignal(pid, syscall.SIGKILL)
	fmt.Printf("SIGKILL sent to pid %v\n", pid)
	return err
}

func NewProcessSlayer() ProcessSlayer {
	return &linuxProcessSlayer{}
}
