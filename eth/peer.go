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
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/teamnsrg/go-ethereum/common"
	"github.com/teamnsrg/go-ethereum/core/types"
	"github.com/teamnsrg/go-ethereum/p2p"
	"github.com/teamnsrg/go-ethereum/rlp"
	"gopkg.in/fatih/set.v0"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

const (
	maxKnownTxs      = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks   = 1024  // Maximum block hashes to keep in the known list (prevent DOS)
	handshakeTimeout = 5 * time.Second
)

// PeerInfo represents a short summary of the Ethereum sub-protocol metadata known
// about a connected peer.
type PeerInfo struct {
	Version    int      `json:"version"`    // Ethereum protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // SHA3 hash of the peer's best owned block
}

type peer struct {
	id string

	*p2p.Peer
	rw p2p.MsgReadWriter

	version  int         // Protocol version negotiated
	forkDrop *time.Timer // Timed connection dropper if forks aren't validated in time

	head common.Hash
	td   *big.Int
	lock sync.RWMutex

	knownTxs    *set.Set // Set of transaction hashes known to be known by this peer
	knownBlocks *set.Set // Set of block hashes known to be known by this peer
}

func newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	id := p.ID()

	return &peer{
		Peer:        p,
		rw:          rw,
		version:     version,
		id:          fmt.Sprintf("%x", id[:8]),
		knownTxs:    set.New(),
		knownBlocks: set.New(),
	}
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *peer) Info() *PeerInfo {
	hash, td := p.Head()

	return &PeerInfo{
		Version:    p.version,
		Difficulty: td,
		Head:       hash.Hex(),
	}
}

// Head retrieves a copy of the current head hash and total difficulty of the
// peer.
func (p *peer) Head() (hash common.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash, new(big.Int).Set(p.td)
}

// SetHead updates the head hash and total difficulty of the peer.
func (p *peer) SetHead(hash common.Hash, td *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head[:], hash[:])
	p.td.Set(td)
}

// MarkBlock marks a block as known for the peer, ensuring that the block will
// never be propagated to this particular peer.
func (p *peer) MarkBlock(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known block hash
	for p.knownBlocks.Size() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash)
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownTxs.Size() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash)
}

// SendTransactions sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
func (p *peer) SendTransactions(txs types.Transactions) error {
	for _, tx := range txs {
		p.knownTxs.Add(tx.Hash())
	}
	return p2p.SendEthSubproto(p.rw, TxMsg, txs, p.ID())
}

// SendNewBlockHashes announces the availability of a number of blocks through
// a hash notification.
func (p *peer) SendNewBlockHashes(hashes []common.Hash, numbers []uint64) error {
	for _, hash := range hashes {
		p.knownBlocks.Add(hash)
	}
	request := make(newBlockHashesData, len(hashes))
	for i := 0; i < len(hashes); i++ {
		request[i].Hash = hashes[i]
		request[i].Number = numbers[i]
	}
	return p2p.SendEthSubproto(p.rw, NewBlockHashesMsg, request, p.ID())
}

// SendNewBlock propagates an entire block to a remote peer.
func (p *peer) SendNewBlock(block *types.Block, td *big.Int) error {
	p.knownBlocks.Add(block.Hash())
	return p2p.SendEthSubproto(p.rw, NewBlockMsg, []interface{}{block, td}, p.ID())
}

// SendBlockHeaders sends a batch of block headers to the remote peer.
func (p *peer) SendBlockHeaders(headers []*types.Header) error {
	return p2p.SendEthSubproto(p.rw, BlockHeadersMsg, headers, p.ID())
}

// SendBlockBodies sends a batch of block contents to the remote peer.
func (p *peer) SendBlockBodies(bodies []*blockBody) error {
	return p2p.SendEthSubproto(p.rw, BlockBodiesMsg, blockBodiesData(bodies), p.ID())
}

// SendBlockBodiesRLP sends a batch of block contents to the remote peer from
// an already RLP encoded format.
func (p *peer) SendBlockBodiesRLP(bodies []rlp.RawValue) error {
	return p2p.SendEthSubproto(p.rw, BlockBodiesMsg, bodies, p.ID())
}

// SendNodeDataRLP sends a batch of arbitrary internal data, corresponding to the
// hashes requested.
func (p *peer) SendNodeData(data [][]byte) error {
	return p2p.SendEthSubproto(p.rw, NodeDataMsg, data, p.ID())
}

// SendReceiptsRLP sends a batch of transaction receipts, corresponding to the
// ones requested from an already RLP encoded format.
func (p *peer) SendReceiptsRLP(receipts []rlp.RawValue) error {
	return p2p.SendEthSubproto(p.rw, ReceiptsMsg, receipts, p.ID())
}

// RequestOneHeader is a wrapper around the header query functions to fetch a
// single header. It is used solely by the fetcher.
func (p *peer) RequestOneHeader(hash common.Hash) error {
	p.Log().Debug("Fetching single header", "hash", hash)
	return p2p.SendEthSubproto(p.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Hash: hash}, Amount: uint64(1), Skip: uint64(0), Reverse: false}, p.ID())
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (p *peer) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool) error {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromhash", origin, "skip", skip, "reverse", reverse)
	return p2p.SendEthSubproto(p.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Hash: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse}, p.ID())
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (p *peer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromnum", origin, "skip", skip, "reverse", reverse)
	return p2p.SendEthSubproto(p.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Number: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse}, p.ID())
}

// RequestBodies fetches a batch of blocks' bodies corresponding to the hashes
// specified.
func (p *peer) RequestBodies(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of block bodies", "count", len(hashes))
	return p2p.SendEthSubproto(p.rw, GetBlockBodiesMsg, hashes, p.ID())
}

// RequestNodeData fetches a batch of arbitrary data from a node's known state
// data, corresponding to the specified hashes.
func (p *peer) RequestNodeData(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of state data", "count", len(hashes))
	return p2p.SendEthSubproto(p.rw, GetNodeDataMsg, hashes, p.ID())
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (p *peer) RequestReceipts(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of receipts", "count", len(hashes))
	return p2p.SendEthSubproto(p.rw, GetReceiptsMsg, hashes, p.ID())
}

// statusDataWrapper is the wrapper for statusData and unixTime
type statusDataWrapper struct {
	ReceivedAt *time.Time
	Status     *statusData
	ErrorCode  errCode
}

func (s *statusDataWrapper) isValidIncompatibleStatus() bool {
	return s.ErrorCode == ErrGenesisBlockMismatch || s.ErrorCode == ErrNetworkIdMismatch || s.ErrorCode == ErrProtocolVersionMismatch
}

// Handshake executes the eth protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (p *peer) Handshake(network uint64, td *big.Int, head common.Hash, genesis common.Hash, statusWrapper *statusDataWrapper) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)

	go func() {
		errc <- p2p.SendEthSubproto(p.rw, StatusMsg, &statusData{
			ProtocolVersion: uint32(p.version),
			NetworkId:       network,
			TD:              td,
			CurrentBlock:    head,
			GenesisBlock:    genesis,
		}, p.ID())
	}()
	go func() {
		errc <- p.readStatus(network, statusWrapper, genesis)
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return p2p.DiscReadTimeout
		}
	}
	p.td, p.head = statusWrapper.Status.TD, statusWrapper.Status.CurrentBlock
	return nil
}

func (p *peer) readStatus(network uint64, statusWrapper *statusDataWrapper, genesis common.Hash) (err error) {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	unixTime := float64(msg.ReceivedAt.UnixNano()) / 1000000000
	if msg.Code != StatusMsg {
		p.Log().Proto("<<UNEXPECTED_"+ethCodeToString[msg.Code], "receivedAt", unixTime, "obj", "<OMITTED>", "size", int(msg.Size), "peer", p.ID())
		return errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > ProtocolMaxMsgSize {
		p.Log().Proto("<<TOOLARGE_"+ethCodeToString[msg.Code], "receivedAt", unixTime, "obj", "<OMITTED>", "size", int(msg.Size), "peer", p.ID())
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	var status statusData
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		p.Log().Proto("<<FAIL_"+ethCodeToString[msg.Code], "receivedAt", unixTime, "obj", "<OMITTED>", "size", int(msg.Size), "peer", p.ID())
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}

	p.Log().Proto("<<"+ethCodeToString[msg.Code], "receivedAt", unixTime, "obj", status, "size", int(msg.Size), "peer", p.ID())

	statusWrapper.ReceivedAt = &msg.ReceivedAt
	statusWrapper.Status = &status

	if status.GenesisBlock != genesis {
		statusWrapper.ErrorCode = ErrGenesisBlockMismatch
		return errResp(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock[:8], genesis[:8])
	}
	if status.NetworkId != network {
		statusWrapper.ErrorCode = ErrNetworkIdMismatch
		return errResp(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, network)
	}
	if int(status.ProtocolVersion) != p.version {
		statusWrapper.ErrorCode = ErrProtocolVersionMismatch
		return errResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
	}
	return nil
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer %s [%s]", p.id,
		fmt.Sprintf("eth/%2d", p.version),
	)
}

// peerSet represents the collection of active peers currently participating in
// the Ethereum sub-protocol.
type peerSet struct {
	peers  map[string]*peer
	lock   sync.RWMutex
	closed bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[p.id]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p
	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[id]; !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// PeersWithoutBlock retrieves a list of peers that do not have a given block in
// their set of known hashes.
func (ps *peerSet) PeersWithoutBlock(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownBlocks.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) PeersWithoutTx(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownTxs.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *peerSet) BestPeer() *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var (
		bestPeer *peer
		bestTd   *big.Int
	)
	for _, p := range ps.peers {
		if _, td := p.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			bestPeer, bestTd = p, td
		}
	}
	return bestPeer
}

// Close disconnects all peers.
// No new peers can be registered after Close has returned.
func (ps *peerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.Disconnect(p2p.DiscQuitting)
	}
	ps.closed = true
}
