package internal

type linuxProcessFinder struct{}

func (pf *linuxProcessFinder) FindPIDByPort(port int) (int, error) {
	return 0, nil
}

func NewProcessFinder() ProcessFinder {
	return &linuxProcessFinder{}
}