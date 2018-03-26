// +build darwin

package p2p

import (
	"net"
	"syscall"
	"unsafe"

	"github.com/teamnsrg/go-ethereum/tcpinfo"
)

// Srtt in seconds (originally milliseconds)
func (p *Peer) Srtt() float64 {
	tcpInfo := p.GetTCPInfo()
	if tcpInfo == nil {
		return 0.0
	}
	return float64(tcpInfo.Srtt) / 1000
}

func (p *Peer) GetTCPInfo() *tcpinfo.TCPConnectionInfo {
	tcpConn, ok := (p.rw.fd).(*net.TCPConn)
	if !ok {
		return nil
	}
	file, err := tcpConn.File()
	if err != nil {
		return nil
	}
	fd := file.Fd()
	size := tcpinfo.SizeofTCPConnectionInfo
	tcpInfo := tcpinfo.TCPConnectionInfo{}
	_, _, e1 := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(int(fd)), uintptr(syscall.IPPROTO_TCP),
		uintptr(tcpinfo.TCP_CONNECTION_INFO), uintptr(unsafe.Pointer(&tcpInfo)), uintptr(unsafe.Pointer(&size)), 0)
	if e1 != 0 {
		return nil
	}
	return &tcpInfo
}
