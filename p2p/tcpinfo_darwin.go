// +build darwin

package p2p

import "github.com/mikioh/tcpinfo"

// Srtt in seconds
func (tc *tcpConn) updateRtt(info *tcpinfo.Info) {
	if info != nil {
		tc.rtt = float64(info.Sys.SRTT) / 1e9
	}
}
