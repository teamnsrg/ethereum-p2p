// +build darwin

package p2p

import (
	"net"
	"syscall"
	"unsafe"

	"github.com/teamnsrg/go-ethereum/tcpinfo"
)

// Receiver mss in bytes
func (rw *rlpxFrameRW) MssRx() uint32 {
	tcpInfo := rw.GetTCPInfo()
	if tcpInfo == nil {
		return 0
	}
	return tcpInfo.Maxseg
}

// Sender mss in bytes
func (rw *rlpxFrameRW) MssTx() uint32 {
	tcpInfo := rw.GetTCPInfo()
	if tcpInfo == nil {
		return 0
	}
	return tcpInfo.Maxseg
}

// Srtt in seconds (originally milliseconds)
func (rw *rlpxFrameRW) Rtt() float64 {
	tcpInfo := rw.GetTCPInfo()
	if tcpInfo == nil {
		return rw.rtt
	}
	rw.rtt = float64(tcpInfo.Srtt) / 1e3
	return rw.rtt
}

func (rw *rlpxFrameRW) GetTCPInfo() *tcpinfo.TCPConnectionInfo {
	tcpConn, ok := rw.conn.(*net.TCPConn)
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
