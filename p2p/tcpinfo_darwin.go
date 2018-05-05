// +build darwin

package p2p

import "github.com/mikioh/tcpinfo"

// Srtt in seconds
func (tc *tcpConn) updateSrtt(info *tcpinfo.Info) {
	if info != nil {
		tc.srtt = float64(info.Sys.SRTT) / 1e9
	}
}
