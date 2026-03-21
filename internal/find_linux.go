package internal

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var ErrPIDNotFound = errors.New("no owner process found for this port")

type linuxProcessFinder struct{}

// Returns the process' PID holding the port.
func (pf *linuxProcessFinder) FindPIDByPort(port int) (int, error) {
	if os.Getuid() != 0 {
		fmt.Println("warning: not running as root, may miss processes owned by other users")
	}

	netFiles := []string{
		"/proc/net/tcp",
		"/proc/net/tcp6",
		"/proc/net/udp",
		"/proc/net/udp6",
	}

	allInodes := map[int]bool{}

	for _, netFile := range netFiles {
		inodes, err := parseNetFile(netFile, port)
		if err != nil {
			fmt.Printf("error parsing %v: %v\n", netFile, err)
			continue
		}

		for _, inode := range inodes {
			allInodes[inode] = true
		}
	}

	numericProcessIds := []int{}
	procDirEntries, _ := os.ReadDir("/proc")
	for _, procEntry := range procDirEntries {
		if isNumber(procEntry.Name()) {
			numericProcessIds = append(numericProcessIds, mustParseInt(procEntry.Name()))
		}
	}

	for _, pid := range numericProcessIds {
		procFdsPath := fmt.Sprintf("/proc/%d/fd", pid)
		procFds, err := (os.ReadDir(procFdsPath))
		if err != nil {
			continue
		}
		for _, fd := range procFds {
			linkTarget, err := os.Readlink(fmt.Sprintf("%s/%s", procFdsPath, fd.Name()))
			if err != nil {
				continue
			}
			// linkTarget looks like "socket:[12345]"
			var inode int
			if _, err := fmt.Sscanf(linkTarget, "socket:[%d]", &inode); err != nil {
				continue // not a socket fd
			}
			if allInodes[inode] {
				return pid, nil
			}
		}
	}

	return 0, ErrPIDNotFound
}

var targetTCPStates = map[int]bool{
	0x01: true, // ESTABLISHED
	0x02: true, // SYN_SENT
	0x0A: true, // LISTEN
	0x08: true, // CLOSE_WAIT
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
			if _, ok := targetTCPStates[netEntry.st]; !ok {
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

func isNumber(s string) bool {
	_, err := strconv.Atoi(s)
	if err != nil {
		return false
	}

	return true
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

// NewProcessFinder returns an instance of [linuxProcessFinder],
// a type which satisfies the [ProcessFinder] interface.
func NewProcessFinder() ProcessFinder {
	return &linuxProcessFinder{}
}
