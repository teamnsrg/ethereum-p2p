// +build linux

package p2p

import "github.com/mikioh/tcpinfo"

// Srtt in seconds
func (tc *tcpConn) updateRtt(info *tcpinfo.Info) {
	if info != nil {
		tc.rtt = float64(info.RTT) / 1e9
	}
}
