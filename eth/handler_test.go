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

package eth

import (
	"math/big"
	"testing"
	"time"

	"github.com/teamnsrg/go-ethereum/consensus/ethash"
	"github.com/teamnsrg/go-ethereum/core"
	"github.com/teamnsrg/go-ethereum/core/types"
	"github.com/teamnsrg/go-ethereum/core/vm"
	"github.com/teamnsrg/go-ethereum/eth/downloader"
	"github.com/teamnsrg/go-ethereum/ethdb"
	"github.com/teamnsrg/go-ethereum/event"
	"github.com/teamnsrg/go-ethereum/p2p"
	"github.com/teamnsrg/go-ethereum/params"
)

var bigTxGas = new(big.Int).SetUint64(params.TxGas)

// Tests that protocol versions and modes of operations are matched up properly.
func TestProtocolCompatibility(t *testing.T) {
	// Define the compatibility chart
	tests := []struct {
		version    uint
		mode       downloader.SyncMode
		compatible bool
	}{
		{61, downloader.FullSync, true}, {62, downloader.FullSync, true}, {63, downloader.FullSync, true},
		{61, downloader.FastSync, false}, {62, downloader.FastSync, false}, {63, downloader.FastSync, true},
	}
	// Make sure anything we screw up is restored
	backup := ProtocolVersions
	defer func() { ProtocolVersions = backup }()

	// Try all available compatibility configs and check for errors
	for i, tt := range tests {
		ProtocolVersions = []uint{tt.version}

		pm, err := newTestProtocolManager(tt.mode, 0, nil, nil)
		if pm != nil {
			defer pm.Stop()
		}
		if (err == nil && !tt.compatible) || (err != nil && tt.compatible) {
			t.Errorf("test %d: compatibility mismatch: have error %v, want compatibility %v", i, err, tt.compatible)
		}
	}
}

// Tests that post eth protocol handshake, DAO fork-enabled clients also execute
// a DAO "challenge" verifying each others' DAO fork headers to ensure they're on
// compatible chains.
func TestDAOChallengeNoVsNo(t *testing.T)       { testDAOChallenge(t, false, false, false) }
func TestDAOChallengeNoVsPro(t *testing.T)      { testDAOChallenge(t, false, true, false) }
func TestDAOChallengeProVsNo(t *testing.T)      { testDAOChallenge(t, true, false, false) }
func TestDAOChallengeProVsPro(t *testing.T)     { testDAOChallenge(t, true, true, false) }
func TestDAOChallengeNoVsTimeout(t *testing.T)  { testDAOChallenge(t, false, false, true) }
func TestDAOChallengeProVsTimeout(t *testing.T) { testDAOChallenge(t, true, true, true) }

func testDAOChallenge(t *testing.T, localForked, remoteForked bool, timeout bool) {
	// Reduce the DAO handshake challenge timeout
	if timeout {
		defer func(old time.Duration) { daoChallengeTimeout = old }(daoChallengeTimeout)
		daoChallengeTimeout = 500 * time.Millisecond
	}
	// Create a DAO aware protocol manager
	var (
		evmux         = new(event.TypeMux)
		pow           = ethash.NewFaker()
		db, _         = ethdb.NewMemDatabase()
		config        = &params.ChainConfig{DAOForkBlock: big.NewInt(1), DAOForkSupport: localForked}
		gspec         = &core.Genesis{Config: config}
		genesis       = gspec.MustCommit(db)
		blockchain, _ = core.NewBlockChain(db, config, pow, vm.Config{})
	)
	pm, err := NewProtocolManager(config, downloader.FullSync, DefaultConfig.NetworkId, evmux, new(testTxPool), pow, blockchain, db)
	if err != nil {
		t.Fatalf("failed to start test protocol manager: %v", err)
	}
	pm.knownNodeInfos = &p2p.KnownNodeInfos{}
	pm.Start(1000)
	defer pm.Stop()

	// Connect a new peer and check that we receive the DAO challenge
	peer, _ := newTestPeer("peer", eth63, pm, true)
	defer peer.close()

	challenge := &getBlockHeadersData{
		Origin:  hashOrNumber{Number: config.DAOForkBlock.Uint64()},
		Amount:  1,
		Skip:    0,
		Reverse: false,
	}
	if err := p2p.ExpectMsg(peer.app, GetBlockHeadersMsg, challenge); err != nil {
		t.Fatalf("challenge mismatch: %v", err)
	}
	// Create a block to reply to the challenge if no timeout is simulated
	if !timeout {
		blocks, _ := core.GenerateChain(&params.ChainConfig{}, genesis, db, 1, func(i int, block *core.BlockGen) {
			if remoteForked {
				block.SetExtra(params.DAOForkBlockExtra)
			}
		})
		if err := p2p.Send(peer.app, BlockHeadersMsg, []*types.Header{blocks[0].Header()}); err != nil {
			t.Fatalf("failed to answer challenge: %v", err)
		}
		time.Sleep(100 * time.Millisecond) // Sleep to avoid the verification racing with the drops
	} else {
		// Otherwise wait until the test timeout passes
		time.Sleep(daoChallengeTimeout + 500*time.Millisecond)
	}
}
