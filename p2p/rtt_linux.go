// +build linux

package p2p

import (
	"net"
	"syscall"
	"unsafe"

	"github.com/teamnsrg/go-ethereum/tcpinfo"
)

// Srtt in seconds (originally microseconds)
func (c *conn) Srtt() float64 {
	tcpInfo := c.GetTCPInfo()
	if tcpInfo == nil {
		return 0.0
	}
	return float64(tcpInfo.Rtt) / 1000000
}

func (c *conn) GetTCPInfo() *tcpinfo.TCPConnectionInfo {
	tcpConn, ok := c.fd.(*net.TCPConn)
	if !ok {
		return nil
	}
	file, err := tcpConn.File()
	if err != nil {
		return nil
	}
	fd := file.Fd()
	size := tcpinfo.SizeofTCPInfo
	tcpInfo := tcpinfo.TCPInfo{}
	_, _, e1 := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(int(fd)), uintptr(syscall.SOL_TCP),
		uintptr(tcpinfo.TCP_INFO), uintptr(unsafe.Pointer(&tcpInfo)), uintptr(unsafe.Pointer(&size)), 0)
	if e1 != 0 {
		return nil
	}
	return &tcpInfo
}
