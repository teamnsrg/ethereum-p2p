package p2p

import (
	"net"

	"github.com/mikioh/tcp"
	"github.com/mikioh/tcpinfo"
)

type tcpConn struct {
	*tcp.Conn

	receiverMss uint32
	senderMss   uint32
	rtt         float64 // recent rtt
	srtt        float64 // smoothed rtt
}

func (tc *tcpConn) getTCPInfo() *tcpinfo.Info {
	var o tcpinfo.Info
	var b [256]byte
	i, err := tc.Conn.Option(o.Level(), o.Name(), b[:])
	if err != nil {
		return nil
	}
	info, ok := i.(*tcpinfo.Info)
	if !ok {
		return nil
	}
	return info
}

func newTCPConn(conn net.Conn) *tcpConn {
	c, err := tcp.NewConn(conn)
	if err != nil {
		return &tcpConn{}
	}
	tc := &tcpConn{Conn: c}
	info := tc.getTCPInfo()
	if info == nil {
		return tc
	}
	tc.receiverMss = uint32(info.ReceiverMSS)
	tc.senderMss = uint32(info.SenderMSS)
	tc.updateRtt(info)
	tc.updateSrtt(info)
	return tc
}
