// +build linux

package p2p

import "github.com/mikioh/tcpinfo"

// Srtt in seconds
func (tc *tcpConn) updateSrtt(info *tcpinfo.Info) {
	if info != nil {
		tc.srtt = float64(info.RTT) / 1e9
	}
}
