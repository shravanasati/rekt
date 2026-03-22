package internal

import (
	"fmt"
	"sort"
	"syscall"
	"unsafe"
)

var ErrPIDNotFound = fmt.Errorf("no process owner found for this port")

const (
	afInet  = 2
	afInet6 = 23

	tcpTableOwnerPIDAll = 5
	udpTableOwnerPID    = 1

	noError                 = 0
	errorInsufficientBuffer = 122
)

var (
	iphlpapi                = syscall.NewLazyDLL("iphlpapi.dll")
	procGetExtendedTCPTable = iphlpapi.NewProc("GetExtendedTcpTable")
	procGetExtendedUDPTable = iphlpapi.NewProc("GetExtendedUdpTable")
)

type mibTCPRowOwnerPID struct {
	state      uint32
	localAddr  uint32
	localPort  uint32
	remoteAddr uint32
	remotePort uint32
	owningPID  uint32
}

type mibTCP6RowOwnerPID struct {
	localAddr     [16]byte
	localScopeID  uint32
	localPort     uint32
	remoteAddr    [16]byte
	remoteScopeID uint32
	remotePort    uint32
	state         uint32
	owningPID     uint32
}

type mibUDPRowOwnerPID struct {
	localAddr uint32
	localPort uint32
	owningPID uint32
}

type mibUDP6RowOwnerPID struct {
	localAddr    [16]byte
	localScopeID uint32
	localPort    uint32
	owningPID    uint32
}

type windowsProcessFinder struct{}

func (pf *windowsProcessFinder) FindPIDByPort(port int) ([]int, error) {
	foundPIDs := map[int]struct{}{}

	collectors := []func(int, map[int]struct{}) error{
		collectTCP4PIDs,
		collectTCP6PIDs,
		collectUDP4PIDs,
		collectUDP6PIDs,
	}

	for _, collect := range collectors {
		if err := collect(port, foundPIDs); err != nil {
			return nil, err
		}
	}

	if len(foundPIDs) == 0 {
		return nil, ErrPIDNotFound
	}

	pids := make([]int, 0, len(foundPIDs))
	for pid := range foundPIDs {
		pids = append(pids, pid)
	}
	sort.Ints(pids)

	return pids, nil
}

func collectTCP4PIDs(targetPort int, foundPIDs map[int]struct{}) error {
	table, err := getExtendedTable(procGetExtendedTCPTable, afInet, tcpTableOwnerPIDAll)
	if err != nil {
		return err
	}
	if len(table) < 4 {
		return nil
	}

	entryCount := *(*uint32)(unsafe.Pointer(&table[0]))
	rowSize := unsafe.Sizeof(mibTCPRowOwnerPID{})
	base := uintptr(unsafe.Pointer(&table[0])) + unsafe.Sizeof(entryCount)

	for i := uint32(0); i < entryCount; i++ {
		row := (*mibTCPRowOwnerPID)(unsafe.Pointer(base + uintptr(i)*rowSize))
		if dwordPortToHostOrder(row.localPort) == targetPort {
			foundPIDs[int(row.owningPID)] = struct{}{}
		}
	}

	return nil
}

func collectTCP6PIDs(targetPort int, foundPIDs map[int]struct{}) error {
	table, err := getExtendedTable(procGetExtendedTCPTable, afInet6, tcpTableOwnerPIDAll)
	if err != nil {
		return err
	}
	if len(table) < 4 {
		return nil
	}

	entryCount := *(*uint32)(unsafe.Pointer(&table[0]))
	rowSize := unsafe.Sizeof(mibTCP6RowOwnerPID{})
	base := uintptr(unsafe.Pointer(&table[0])) + unsafe.Sizeof(entryCount)

	for i := uint32(0); i < entryCount; i++ {
		row := (*mibTCP6RowOwnerPID)(unsafe.Pointer(base + uintptr(i)*rowSize))
		if dwordPortToHostOrder(row.localPort) == targetPort {
			foundPIDs[int(row.owningPID)] = struct{}{}
		}
	}

	return nil
}

func collectUDP4PIDs(targetPort int, foundPIDs map[int]struct{}) error {
	table, err := getExtendedTable(procGetExtendedUDPTable, afInet, udpTableOwnerPID)
	if err != nil {
		return err
	}
	if len(table) < 4 {
		return nil
	}

	entryCount := *(*uint32)(unsafe.Pointer(&table[0]))
	rowSize := unsafe.Sizeof(mibUDPRowOwnerPID{})
	base := uintptr(unsafe.Pointer(&table[0])) + unsafe.Sizeof(entryCount)

	for i := uint32(0); i < entryCount; i++ {
		row := (*mibUDPRowOwnerPID)(unsafe.Pointer(base + uintptr(i)*rowSize))
		if dwordPortToHostOrder(row.localPort) == targetPort {
			foundPIDs[int(row.owningPID)] = struct{}{}
		}
	}

	return nil
}

func collectUDP6PIDs(targetPort int, foundPIDs map[int]struct{}) error {
	table, err := getExtendedTable(procGetExtendedUDPTable, afInet6, udpTableOwnerPID)
	if err != nil {
		return err
	}
	if len(table) < 4 {
		return nil
	}

	entryCount := *(*uint32)(unsafe.Pointer(&table[0]))
	rowSize := unsafe.Sizeof(mibUDP6RowOwnerPID{})
	base := uintptr(unsafe.Pointer(&table[0])) + unsafe.Sizeof(entryCount)

	for i := uint32(0); i < entryCount; i++ {
		row := (*mibUDP6RowOwnerPID)(unsafe.Pointer(base + uintptr(i)*rowSize))
		if dwordPortToHostOrder(row.localPort) == targetPort {
			foundPIDs[int(row.owningPID)] = struct{}{}
		}
	}

	return nil
}

func getExtendedTable(proc *syscall.LazyProc, af uint32, tableClass uint32) ([]byte, error) {
	var tableSize uint32

	// Win32 table APIs require a first probe call to discover the right buffer size.
	r1, _, _ := proc.Call(
		0,
		uintptr(unsafe.Pointer(&tableSize)),
		0,
		uintptr(af),
		uintptr(tableClass),
		0,
	)
	if uint32(r1) != noError && uint32(r1) != errorInsufficientBuffer {
		return nil, fmt.Errorf("initial table size query failed (af=%d class=%d): winerr=%d", af, tableClass, r1)
	}

	if tableSize == 0 {
		return []byte{}, nil
	}

	table := make([]byte, tableSize)
	r1, _, _ = proc.Call(
		uintptr(unsafe.Pointer(&table[0])),
		uintptr(unsafe.Pointer(&tableSize)),
		0,
		uintptr(af),
		uintptr(tableClass),
		0,
	)
	if uint32(r1) != noError {
		return nil, fmt.Errorf("table fetch failed (af=%d class=%d): winerr=%d", af, tableClass, r1)
	}

	return table, nil
}

func dwordPortToHostOrder(dwordPort uint32) int {
	networkPort := uint16(dwordPort)
	hostPort := (networkPort >> 8) | (networkPort << 8)
	return int(hostPort)
}

func NewProcessFinder() ProcessFinder {
	return &windowsProcessFinder{}
}
