package internal

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var ErrPIDNotFound = errors.New("no process owner found for this port")

const maxPIDScanWorkers = 16

type linuxProcessFinder struct{}

// Returns the process' PID holding the port.
func (pf *linuxProcessFinder) FindPIDByPort(port int, verbose bool) ([]*Process, error) {
	if os.Getuid() != 0 {
		fmt.Println("warning: not running as root, may miss processes owned by other users")
	}

	netFiles := []string{
		"/proc/net/tcp",
		"/proc/net/tcp6",
		"/proc/net/udp",
		"/proc/net/udp6",
	}

	allInodes := getActiveInodes(netFiles, port)
	if len(allInodes) == 0 {
		return nil, ErrPIDNotFound
	}

	numericProcessIds := getNumericPIDs()
	if len(numericProcessIds) == 0 {
		return nil, ErrPIDNotFound
	}

	workerCount := runtime.GOMAXPROCS(0)
	if workerCount > maxPIDScanWorkers {
		workerCount = maxPIDScanWorkers
	}
	if workerCount > len(numericProcessIds) {
		workerCount = len(numericProcessIds)
	}
	if workerCount < 1 {
		workerCount = 1
	}

	jobs := make(chan int)
	results := make(chan *Process, workerCount)

	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for pid := range jobs {
			owned, netfile := pidOwnsAnyTargetInode(pid, allInodes)
			if owned {
				netfile, _ = strings.CutPrefix(netfile, "/proc/net/")
				results <- &Process{PID: pid, Type: netfile}
			}
		}
	}

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for _, pid := range numericProcessIds {
		jobs <- pid
	}
	close(jobs)

	processes := []*Process{}
	for process := range results {
		if verbose {
			readProcessInfo(process)
		}
		processes = append(processes, process)
	}

	if len(processes) > 0 {
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].PID < processes[j].PID
		})
		return processes, nil
	}

	return nil, ErrPIDNotFound
}

func readProcessInfo(process *Process) {
	statusFile, err := os.Open("/proc/" + strconv.Itoa(process.PID) + "/status")
	if err != nil {
		fmt.Printf("error reading /proc/%d/status: %v\n", process.PID, err)
		return
	}
	defer statusFile.Close()

	// read name
	reader := bufio.NewReader(statusFile)
	foundName, foundUser := false, false
	for !foundName || !foundUser {
		key, err := reader.ReadString(':')
		if err != nil && err != io.EOF {
			fmt.Printf("error reading /proc/%d/status: %v\n", process.PID, err)
			return
		}
		value, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("error reading /proc/%d/status: %v\n", process.PID, err)
			return
		}
		key = key[:len(key)-1]
		value = value[:len(value)-1]
		// fmt.Printf("key=`%v`\n", key)
		// fmt.Printf("value=`%v`\n", value)

		switch key {
		case "Name":
			foundName = true
			process.Name = strings.TrimSpace(value)
		case "Uid":
			foundUser = true
			usr, err := user.LookupId(parseUidString(value))
			if err != nil {
				fmt.Printf("error looking up uid=%v: %v\n", value, err)
				return
			}
			process.User = usr.Name
		}

	}
}

func parseUidString(s string) string {
	// looks like
	// ` 1000 1000 1000`
	uid := ""
	numberStarted := false
	for _, v := range s {
		isNotNumber := v < '0' || v > '9'
		if isNotNumber && !numberStarted {
			continue
		}
		if isNotNumber {
			return uid
		}
		numberStarted = true
		uid += string(v)
	}
	return uid
}

func pidOwnsAnyTargetInode(pid int, targetInodes map[int]string) (bool, string) {
	procFdsPath := "/proc/" + strconv.Itoa(pid) + "/fd"
	procFds, err := os.ReadDir(procFdsPath)
	if err != nil {
		return false, ""
	}

	for _, fd := range procFds {
		fdPath := procFdsPath + "/" + fd.Name()
		linkTarget, err := os.Readlink(fdPath)
		if err != nil {
			continue
		}
		inode, ok := parseSocketInode(linkTarget)
		if !ok {
			continue // not a socket fd
		}
		if val, ok := targetInodes[inode]; ok {
			return true, val
		}
	}

	return false, ""
}

func getActiveInodes(netFiles []string, port int) map[int]string {
	allInodes := make(map[int]string, 0)

	for _, netFile := range netFiles {
		inodes, err := parseNetFile(netFile, port)
		if err != nil {
			fmt.Printf("error parsing %v: %v\n", netFile, err)
			continue
		}

		for _, inode := range inodes {
			allInodes[inode] = netFile
		}
	}

	return allInodes
}

func getNumericPIDs() []int {
	numericProcessIds := []int{}
	procDirEntries, err := os.ReadDir("/proc")
	if err != nil {
		fmt.Printf("error reading /proc: %v\n", err)
		return []int{}
	}
	for _, procEntry := range procDirEntries {
		n, err := strconv.Atoi(procEntry.Name())
		if err == nil {
			numericProcessIds = append(numericProcessIds, n)
		}
	}
	return numericProcessIds
}

// parses an entire net file and returns all matching inodes.
func parseNetFile(netfilepath string, port int) ([]int, error) {
	netFile, err := os.Open(netfilepath)
	if err != nil {
		return []int{}, err
	}
	defer netFile.Close()

	isTcpFile := strings.Contains(netfilepath, "tcp")
	matchingInodes := []int{}

	scanner := bufio.NewScanner(netFile)
	isHeaderLine := true
	for scanner.Scan() {
		line := scanner.Text()
		if isHeaderLine {
			isHeaderLine = false
			continue
		}
		netEntry := parseNetLine(line)
		if isTcpFile {
			// only target certain states in the tcp table
			shouldContinue := true
			switch netEntry.st {
			// ESTABLISHED, SYN_SENT, LISTEN, CLOSE_WAIT
			case 0x01, 0x02, 0x0A, 0x08:
				shouldContinue = false
			}
			if shouldContinue {
				continue
			}
		}

		if port == netEntry.port {
			matchingInodes = append(matchingInodes, netEntry.inode)
		}
	}

	if err := scanner.Err(); err != nil {
		return []int{}, err
	}

	return matchingInodes, nil
}

// netLineEntry represents a line entry from a net file. only relevant fields are stored.
type netLineEntry struct {
	local_address string
	port          int
	st            int
	inode         int
}

// converts a hex string to integer. panics if it fails. choosing panic behavior because
// we can assume the kernel always returns valid hex strings
func mustHexToInt(n string) int {
	converted, err := strconv.ParseInt(n, 16, 0)
	if err != nil {
		panic(err)
	}
	return int(converted)
}

// converts a string to integer. panics if it fails. choosing panic behavior because
// we can assume the kernel always returns valid numeric strings
func mustParseInt(n string) int {
	converted, err := strconv.Atoi(n)
	if err != nil {
		panic(err)
	}

	return converted
}

// parses an individual line from a net file.
func parseNetLine(line string) *netLineEntry {
	splitted := strings.Fields(line)

	local_address := splitted[1]

	portStr := strings.Split(local_address, ":")[1]
	port := mustHexToInt(portStr)

	stateStr := splitted[3]
	state := mustHexToInt(stateStr)

	inodeStr := splitted[9]
	inode := mustParseInt(inodeStr)

	return &netLineEntry{
		local_address: local_address,
		port:          port,
		st:            state,
		inode:         inode,
	}
}

func parseSocketInode(linkTarget string) (int, bool) {
	const socketPrefix = "socket:["
	if !strings.HasPrefix(linkTarget, socketPrefix) || !strings.HasSuffix(linkTarget, "]") {
		return 0, false
	}

	inodeStr := linkTarget[len(socketPrefix) : len(linkTarget)-1]
	inode, err := strconv.Atoi(inodeStr)
	if err != nil {
		return 0, false
	}

	return inode, true
}

// NewProcessFinder returns an instance of [linuxProcessFinder],
// a type which satisfies the [ProcessFinder] interface.
func NewProcessFinder() ProcessFinder {
	return &linuxProcessFinder{}
}
