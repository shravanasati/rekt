package internal

import (
	"fmt"
	"golang.org/x/sys/windows"
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

func (pf *windowsProcessFinder) FindPIDByPort(port int, verbose bool) ([]*Process, error) {
	// map of pid -> type (udp/udp6/tcp/tcp6)
	foundPIDs := map[int]string{}

	collectors := []func(int, map[int]string) error{
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

	processes := make([]*Process, 0, len(foundPIDs))
	for pid := range foundPIDs {
		pr := &Process{PID: pid, Type: foundPIDs[pid]}
		if verbose {
			readProcessInfo(pr)
		}
		processes = append(processes, pr)
	}
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].PID < processes[j].PID
	})

	return processes, nil
}

func readProcessInfo(pr *Process) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pr.PID))
	if err != nil {
		fmt.Printf("error in opening process: %v\n", err)
		return
	}
	defer windows.CloseHandle(handle)

	// executable path
	buf := make([]uint16, windows.MAX_PATH)
	exeSize := uint32(len(buf))
	err = windows.QueryFullProcessImageName(handle, 0, &buf[0], &exeSize)
	if err == nil {
		pr.Name = windows.UTF16ToString(buf[:exeSize])
	} else {
		fmt.Printf("error in query process image name: %v\n", err)
	}

	// owner
	var token windows.Token
	if err := windows.OpenProcessToken(handle, windows.TOKEN_QUERY, &token); err == nil {
		defer token.Close()
		if user, err := token.GetTokenUser(); err == nil {
			account, _, _, err := user.User.Sid.LookupAccount("")
			if err == nil {
				pr.User = account
			}
		}
	}
}

func collectTCP4PIDs(targetPort int, foundPIDs map[int]string) error {
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
			foundPIDs[int(row.owningPID)] = "tcp"
		}
	}

	return nil
}

func collectTCP6PIDs(targetPort int, foundPIDs map[int]string) error {
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
			foundPIDs[int(row.owningPID)] = "tcp6"
		}
	}

	return nil
}

func collectUDP4PIDs(targetPort int, foundPIDs map[int]string) error {
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
			foundPIDs[int(row.owningPID)] = "udp"
		}
	}

	return nil
}

func collectUDP6PIDs(targetPort int, foundPIDs map[int]string) error {
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
			foundPIDs[int(row.owningPID)] = "udp6"
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
