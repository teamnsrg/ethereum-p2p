// Copyright 2015 The go-ethereum Authors
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
	"encoding/binary"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/teamnsrg/go-ethereum/p2p/discover"
	"github.com/teamnsrg/go-ethereum/p2p/netutil"
)

func init() {
	spew.Config.Indent = "\t"
}

type dialtest struct {
	init   *dialstate // state before and after the test.
	rounds []round
}

type round struct {
	peers []*Peer // current peer set
	done  []task  // tasks that got done this round
	new   []task  // the result must match this one
}

func runDialTest(t *testing.T, test dialtest) {
	var (
		vtime   time.Time
		running int
	)
	pm := func(ps []*Peer) map[discover.NodeID]*Peer {
		m := make(map[discover.NodeID]*Peer)
		for _, p := range ps {
			m[p.rw.id] = p
		}
		return m
	}
	for i, round := range test.rounds {
		for _, task := range round.done {
			running--
			if running < 0 {
				panic("running task counter underflow")
			}
			test.init.taskDone(task, vtime, make(map[discover.NodeID]*Peer))
		}

		new := test.init.newTasks(running, len(round.new), pm(round.peers), vtime)
		for _, t := range new {
			if t, ok := t.(*dialTask); ok {
				t.lastSuccess = time.Time{}
			}
		}
		if !sametasks(new, round.new) {
			t.Errorf("round %d: new tasks mismatch:\ngot %v\nwant %v\nstate: %v\nrunning: %v\n",
				i, spew.Sdump(new), spew.Sdump(round.new), spew.Sdump(test.init), spew.Sdump(running))
		}

		// Time advances by 16 seconds on every round.
		vtime = vtime.Add(16 * time.Second)
		running += len(new)
	}
}

type fakeTable []*discover.Node

func (t fakeTable) Self() *discover.Node                     { return new(discover.Node) }
func (t fakeTable) Close()                                   {}
func (t fakeTable) Lookup(discover.NodeID) []*discover.Node  { return nil }
func (t fakeTable) Resolve(discover.NodeID) *discover.Node   { return nil }
func (t fakeTable) ReadRandomNodes(buf []*discover.Node) int { return copy(buf, t) }

// This test checks that candidates that do not match the netrestrict list are not dialed.
func TestDialStateNetRestrict(t *testing.T) {
	// This table always returns the same random nodes
	// in the order given below.
	static := []*discover.Node{
		{ID: uintID(1), IP: net.ParseIP("127.0.0.1")},
		{ID: uintID(2), IP: net.ParseIP("127.0.0.2")},
		{ID: uintID(3), IP: net.ParseIP("127.0.0.3")},
		{ID: uintID(4), IP: net.ParseIP("127.0.0.4")},
		{ID: uintID(5), IP: net.ParseIP("127.0.2.5")},
		{ID: uintID(6), IP: net.ParseIP("127.0.2.6")},
	}
	restrict := new(netutil.Netlist)
	restrict.Add("127.0.2.0/24")

	dialer := newDialState(static, fakeTable{}, restrict)
	dialer.redialFreq = 30 * time.Second
	runDialTest(t, dialtest{
		init: dialer,
		rounds: []round{
			{
				new: []task{
					&dialTask{flags: staticDialedConn, dest: static[4]},
					&dialTask{flags: staticDialedConn, dest: static[5]},
				},
			},
		},
	})
}

// This test checks that static dials are launched.
func TestDialStateStaticDial(t *testing.T) {
	wantStatic := []*discover.Node{
		{ID: uintID(1)},
		{ID: uintID(2)},
		{ID: uintID(3)},
		{ID: uintID(4)},
		{ID: uintID(5)},
	}

	dialer := newDialState(wantStatic, fakeTable{}, nil)
	dialer.redialFreq = 30 * time.Second
	runDialTest(t, dialtest{
		init: dialer,
		rounds: []round{
			// Static dials are launched for the nodes that
			// aren't yet connected.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(1)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(2)}},
				},
				new: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(3)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(5)}},
				},
			},
			// No new tasks are launched in this round because all static
			// nodes are either connected or still being dialed.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(1)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(2)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(3)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(3)}},
				},
			},
			// No new dial tasks are launched because all static
			// nodes are now connected.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(1)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(2)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(3)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(4)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(5)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(5)}},
				},
				new: []task{
					&waitExpireTask{Duration: 14 * time.Second},
				},
			},
			// Wait a round for dial history to expire, no new tasks should spawn.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(1)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(2)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(3)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(4)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(5)}},
				},
			},
			// If a static node is dropped, it should be immediately redialed,
			// irrespective whether it was originally static or dynamic.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(1)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(3)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(5)}},
				},
				new: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(2)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(4)}},
				},
			},
		},
	})
}

// This test checks that past dials are not retried for some time.
func TestDialStateCache(t *testing.T) {
	wantStatic := []*discover.Node{
		{ID: uintID(1)},
		{ID: uintID(2)},
		{ID: uintID(3)},
	}
	dialer := newDialState(wantStatic, fakeTable{}, nil)
	dialer.redialFreq = 30 * time.Second
	runDialTest(t, dialtest{
		init: dialer,
		rounds: []round{
			// Static dials are launched for the nodes that
			// aren't yet connected.
			{
				peers: nil,
				new: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(1)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(2)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(3)}},
				},
			},
			// No new tasks are launched in this round because all static
			// nodes are either connected or still being dialed.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(1)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(2)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(1)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(2)}},
				},
			},
			// A salvage task is launched to wait for node 3's history
			// entry to expire.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(1)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(2)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(3)}},
				},
				new: []task{
					&waitExpireTask{Duration: 14 * time.Second},
				},
			},
			// Still waiting for node 3's entry to expire in the cache.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(1)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(2)}},
				},
			},
			// The cache entry for node 3 has expired and is retried.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(1)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(2)}},
				},
				new: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(3)}},
				},
			},
		},
	})
}

// compares task lists but doesn't care about the order.
func sametasks(a, b []task) bool {
	if len(a) != len(b) {
		return false
	}
next:
	for _, ta := range a {
		for _, tb := range b {
			if reflect.DeepEqual(ta, tb) {
				continue next
			}
		}
		return false
	}
	return true
}

func uintID(i uint32) discover.NodeID {
	var id discover.NodeID
	binary.BigEndian.PutUint32(id[:], i)
	return id
}
