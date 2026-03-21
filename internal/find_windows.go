package internal

type windowsProcessFinder struct{}

func (pf *windowsProcessFinder) FindPIDByPort(port int) (int, error) {
	return 0, nil
}

func NewProcessFinder() ProcessFinder {
	return &windowsProcessFinder{}
}