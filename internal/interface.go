package internal

// Represents a looked up process. Only PID is required. Rest of the fields are populated in verbose field.
type Process struct {
	PID  int
	Name string // executable name
	User string // process owner
	Type string // udp/tcp/tcp6/udp6
}

type ProcessFinder interface {
	FindPIDByPort(port int, verbose bool) ([]*Process, error)
}

type KillMode int

const (
	ModeTerm KillMode = iota
	ModeKill
)

type ProcessSlayer interface {
	KillProcess(pid int) error
	TermProcess(pid int) error
}
