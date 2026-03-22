package internal

import (
	"fmt"
	"syscall"
)

type windowsProcessSlayer struct{}

func (ps *windowsProcessSlayer) KillProcess(pid int) error {
	handle, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(handle)

	if err := syscall.TerminateProcess(handle, 1); err != nil {
		return err
	}

	fmt.Printf("TerminateProcess sent to pid %v\n", pid)
	return nil
}

func (ps *windowsProcessSlayer) TermProcess(pid int) error {
	return ps.KillProcess(pid)
}

func NewProcessSlayer() ProcessSlayer {
	return &windowsProcessSlayer{}
}
