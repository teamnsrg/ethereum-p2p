// +build linux

package p2p

import "github.com/mikioh/tcpinfo"

// recent rtt in seconds
func (tc *tcpConn) updateRtt(info *tcpinfo.Info) {
	if info != nil {
		tc.rtt = float64(info.Sys.ReceiverRTT) / 1e9
	}
}

// Srtt in seconds
func (tc *tcpConn) updateSrtt(info *tcpinfo.Info) {
	if info != nil {
		tc.srtt = float64(info.RTT) / 1e9
	}
}
