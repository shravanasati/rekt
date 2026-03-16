package internal

type ProcessFinder interface {
	FindPIDByPort(port int) (int, error)
}