package network

import (
	"os"
	"syscall"
)

// Socket File method for conn and listener
type Socket interface {
	File() (*os.File, error)
}

// SetTransparentOpt set tranparent option for socket
func SetTransparentOpt(s Socket) error {
	cs, err := s.File()
	if err != nil {
		return err
	}

	defer cs.Close()

	return syscall.SetsockoptInt(int(cs.Fd()), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1)
}
