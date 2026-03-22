package internal

type ProcessFinder interface {
	FindPIDByPort(port int) ([]int, error)
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
