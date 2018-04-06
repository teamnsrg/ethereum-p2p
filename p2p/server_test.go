// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package p2p

import (
	"crypto/ecdsa"
	"errors"
	"math/rand"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/teamnsrg/go-ethereum/crypto"
	"github.com/teamnsrg/go-ethereum/crypto/sha3"
	"github.com/teamnsrg/go-ethereum/p2p/discover"
	"github.com/teamnsrg/go-ethereum/p2p/netutil"
)

func init() {
	// log.Root().SetHandler(log.LvlFilterHandler(log.LvlError, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}

type testTransport struct {
	id discover.NodeID
	*rlpx

	closeErr error
}

func newTestTransport(id discover.NodeID, fd net.Conn) transport {
	wrapped := newRLPX(fd).(*rlpx)
	wrapped.rw = newRLPXFrameRW(fd, secrets{
		MAC:        zero16,
		AES:        zero16,
		IngressMAC: sha3.NewKeccak256(),
		EgressMAC:  sha3.NewKeccak256(),
	})
	return &testTransport{id: id, rlpx: wrapped}
}

func (c *testTransport) Rtt() float64 {
	return 0.0
}

func (c *testTransport) doEncHandshake(prv *ecdsa.PrivateKey, dialDest *discover.Node) (discover.NodeID, error) {
	return c.id, nil
}

func (c *testTransport) doProtoHandshake(our *protoHandshake) (*protoHandshake, Msg, error) {
	return &protoHandshake{ID: c.id, Name: "test"}, Msg{}, nil
}

func (c *testTransport) close(err error) {
	c.rlpx.fd.Close()
	c.closeErr = err
}

func startTestServer(t *testing.T, id discover.NodeID, pf func(*Peer)) *Server {
	config := Config{
		Name:           "test",
		MaxPeers:       10,
		ListenAddr:     "127.0.0.1:0",
		PrivateKey:     newkey(),
		MaxAcceptConns: 50,
	}
	server := &Server{
		Config:       config,
		newPeerHook:  pf,
		newTransport: func(fd net.Conn) transport { return newTestTransport(id, fd) },
	}
	if err := server.Start(); err != nil {
		t.Fatalf("Could not start server: %v", err)
	}
	return server
}

func TestServerListen(t *testing.T) {
	// start the test server
	connected := make(chan *Peer)
	remid := randomID()
	srv := startTestServer(t, remid, func(p *Peer) {
		if p.ID() != remid {
			t.Error("peer func called with wrong node id")
		}
		if p == nil {
			t.Error("peer func called with nil conn")
		}
		connected <- p
	})
	defer close(connected)
	defer srv.Stop()

	// dial the test server
	conn, err := net.DialTimeout("tcp", srv.ListenAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}
	defer conn.Close()

	select {
	case peer := <-connected:
		if peer.LocalAddr().String() != conn.RemoteAddr().String() {
			t.Errorf("peer started with wrong conn: got %v, want %v",
				peer.LocalAddr(), conn.RemoteAddr())
		}
		peers := srv.Peers()
		if !reflect.DeepEqual(peers, []*Peer{peer}) {
			t.Errorf("Peers mismatch: got %v, want %v", peers, []*Peer{peer})
		}
	case <-time.After(1 * time.Second):
		t.Error("server did not accept within one second")
	}
}

func TestServerDial(t *testing.T) {
	// run a one-shot TCP server to handle the connection.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not setup listener: %v", err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Error("accept error:", err)
			return
		}
		accepted <- conn
	}()

	// start the server
	connected := make(chan *Peer)
	remid := randomID()
	srv := startTestServer(t, remid, func(p *Peer) { connected <- p })
	defer close(connected)
	defer srv.Stop()

	// tell the server to connect
	tcpAddr := listener.Addr().(*net.TCPAddr)
	srv.AddPeer(&discover.Node{ID: remid, IP: tcpAddr.IP, TCP: uint16(tcpAddr.Port)})

	select {
	case conn := <-accepted:
		defer conn.Close()

		select {
		case peer := <-connected:
			if peer.ID() != remid {
				t.Errorf("peer has wrong id")
			}
			if peer.Name() != "test" {
				t.Errorf("peer has wrong name")
			}
			if peer.RemoteAddr().String() != conn.LocalAddr().String() {
				t.Errorf("peer started with wrong conn: got %v, want %v",
					peer.RemoteAddr(), conn.LocalAddr())
			}
			peers := srv.Peers()
			if !reflect.DeepEqual(peers, []*Peer{peer}) {
				t.Errorf("Peers mismatch: got %v, want %v", peers, []*Peer{peer})
			}
		case <-time.After(1 * time.Second):
			t.Error("server did not launch peer within one second")
		}

	case <-time.After(1 * time.Second):
		t.Error("server did not connect within one second")
	}
}

// This test checks that Server doesn't drop tasks,
// even if newTasks returns more than the maximum number of tasks.
func TestServerManyTasks(t *testing.T) {
	alltasks := make([]task, 300)
	for i := range alltasks {
		alltasks[i] = &testTask{index: i}
	}

	var (
		srv  = &Server{quit: make(chan struct{}), running: true}
		done = make(chan *testTask)
	)
	srv.MaxAcceptConns = 50
	defer srv.Stop()
	srv.loopWG.Add(1)
	go srv.run(taskgen{
		newFunc: func(running int, peers map[discover.NodeID]*Peer) []task {
			return alltasks
		},
		doneFunc: func(tt task) {
			done <- tt.(*testTask)
		},
	})

	doneset := make(map[int]bool)
	timeout := time.After(2 * time.Second)
	for len(doneset) < len(alltasks) {
		select {
		case tt := <-done:
			if doneset[tt.index] {
				t.Errorf("task %d got done more than once", tt.index)
			} else {
				doneset[tt.index] = true
			}
		case <-timeout:
			t.Errorf("%d of %d tasks got done within 2s", len(doneset), len(alltasks))
			for i := 0; i < len(alltasks); i++ {
				if !doneset[i] {
					t.Logf("task %d not done", i)
				}
			}
			return
		}
	}
}

type taskgen struct {
	newFunc  func(running int, peers map[discover.NodeID]*Peer) []task
	doneFunc func(task)
}

func (tg taskgen) newTasks(running int, peers map[discover.NodeID]*Peer, now time.Time) []task {
	return tg.newFunc(running, peers)
}
func (tg taskgen) taskDone(t task, now time.Time) {
	tg.doneFunc(t)
}
func (tg taskgen) addStatic(*discover.Node) *dialTask {
	return nil
}
func (tg taskgen) removeStatic(*discover.Node) {
}
func (tg taskgen) GetDialFreq() time.Duration {
	return 0
}
func (tg taskgen) SetDialFreq(f int) {
}
func (tg taskgen) GetBlacklist() *netutil.Netlist {
	return nil
}
func (tg taskgen) SetBlacklist(blacklist *netutil.Netlist) {
}

type testTask struct {
	index  int
	called bool
}

func (t *testTask) Do(srv *Server) {
	t.called = true
}

func TestServerSetupConn(t *testing.T) {
	id := randomID()
	srvkey := newkey()
	srvid := discover.PubkeyID(&srvkey.PublicKey)
	tests := []struct {
		dontstart bool
		tt        *setupTransport
		flags     connFlag
		dialDest  *discover.Node

		wantCloseErr error
		wantCalls    string
	}{
		{
			dontstart:    true,
			tt:           &setupTransport{id: id},
			wantCalls:    "close,",
			wantCloseErr: errServerStopped,
		},
		{
			tt:           &setupTransport{id: id, encHandshakeErr: errors.New("read error")},
			flags:        inboundConn,
			wantCalls:    "doEncHandshake,close,",
			wantCloseErr: errors.New("read error"),
		},
		{
			tt:           &setupTransport{id: id},
			dialDest:     &discover.Node{ID: randomID()},
			flags:        staticDialedConn,
			wantCalls:    "doEncHandshake,close,",
			wantCloseErr: DiscUnexpectedIdentity,
		},
		{
			tt:           &setupTransport{id: id, phs: &protoHandshake{ID: randomID()}},
			dialDest:     &discover.Node{ID: id},
			flags:        staticDialedConn,
			wantCalls:    "doEncHandshake,doProtoHandshake,close,",
			wantCloseErr: DiscUnexpectedIdentity,
		},
		{
			tt:           &setupTransport{id: id, protoHandshakeErr: errors.New("foo")},
			dialDest:     &discover.Node{ID: id},
			flags:        staticDialedConn,
			wantCalls:    "doEncHandshake,doProtoHandshake,close,",
			wantCloseErr: errors.New("foo"),
		},
		{
			tt:           &setupTransport{id: srvid, phs: &protoHandshake{ID: srvid}},
			flags:        inboundConn,
			wantCalls:    "doEncHandshake,close,",
			wantCloseErr: DiscSelf,
		},
		{
			tt:           &setupTransport{id: id, phs: &protoHandshake{ID: id}},
			flags:        inboundConn,
			wantCalls:    "doEncHandshake,doProtoHandshake,close,",
			wantCloseErr: DiscUselessPeer,
		},
	}

	for i, test := range tests {
		srv := &Server{
			Config: Config{
				PrivateKey: srvkey,
				MaxPeers:   10,
				Protocols:  []Protocol{discard},
			},
			newTransport: func(fd net.Conn) transport { return test.tt },
		}
		if !test.dontstart {
			if err := srv.Start(); err != nil {
				t.Fatalf("couldn't start server: %v", err)
			}
		}
		p1, _ := net.Pipe()
		srv.SetupConn(p1, test.flags, test.dialDest)
		if !reflect.DeepEqual(test.tt.closeErr, test.wantCloseErr) {
			t.Errorf("test %d: close error mismatch: got %q, want %q", i, test.tt.closeErr, test.wantCloseErr)
		}
		if test.tt.calls != test.wantCalls {
			t.Errorf("test %d: calls mismatch: got %q, want %q", i, test.tt.calls, test.wantCalls)
		}
	}
}

type setupTransport struct {
	id              discover.NodeID
	encHandshakeErr error

	phs               *protoHandshake
	protoHandshakeErr error

	calls    string
	closeErr error
}

func (c *setupTransport) Rtt() float64 {
	return 0.0
}

func (c *setupTransport) doEncHandshake(prv *ecdsa.PrivateKey, dialDest *discover.Node) (discover.NodeID, error) {
	c.calls += "doEncHandshake,"
	return c.id, c.encHandshakeErr
}
func (c *setupTransport) doProtoHandshake(our *protoHandshake) (*protoHandshake, Msg, error) {
	c.calls += "doProtoHandshake,"
	if c.protoHandshakeErr != nil {
		return nil, Msg{}, c.protoHandshakeErr
	}
	return c.phs, Msg{}, nil
}
func (c *setupTransport) close(err error) {
	c.calls += "close,"
	c.closeErr = err
}

// setupConn shouldn't write to/read from the connection.
func (c *setupTransport) WriteMsg(Msg) error {
	panic("WriteMsg called on setupTransport")
}
func (c *setupTransport) ReadMsg() (Msg, error) {
	panic("ReadMsg called on setupTransport")
}

func newkey() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}
	return key
}

func randomID() (id discover.NodeID) {
	for i := range id {
		id[i] = byte(rand.Intn(255))
	}
	return id
}
