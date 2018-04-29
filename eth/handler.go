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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"

	"github.com/teamnsrg/go-ethereum/common"
	"github.com/teamnsrg/go-ethereum/consensus"
	"github.com/teamnsrg/go-ethereum/consensus/misc"
	"github.com/teamnsrg/go-ethereum/core"
	"github.com/teamnsrg/go-ethereum/core/types"
	"github.com/teamnsrg/go-ethereum/eth/downloader"
	"github.com/teamnsrg/go-ethereum/ethdb"
	"github.com/teamnsrg/go-ethereum/event"
	"github.com/teamnsrg/go-ethereum/log"
	"github.com/teamnsrg/go-ethereum/p2p"
	"github.com/teamnsrg/go-ethereum/p2p/discover"
	"github.com/teamnsrg/go-ethereum/params"
	"github.com/teamnsrg/go-ethereum/rlp"
)

const (
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header
)

var (
	daoChallengeTimeout = 15 * time.Second // Time allowance for a node to reply to the DAO handshake challenge
)

// errIncompatibleConfig is returned if the requested protocols and configs are
// not compatible (low protocol version restrictions and high requirements).
var errIncompatibleConfig = errors.New("incompatible configuration")

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type mapWrapper struct {
	sync.Mutex
	known map[common.Hash]struct{}
}

type ProtocolManager struct {
	noMaxPeers bool // Flag whether to ignore maxPeers

	networkId uint64

	fastSync  uint32 // Flag whether fast sync is enabled (gets disabled if we already have blocks)
	acceptTxs uint32 // Flag whether we're considered synchronised (enables transaction processing)

	knownTxs    mapWrapper // All transactions to allow lookups
	knownBlocks mapWrapper // All blocks to allow lookups
	blockchain  *core.BlockChain
	chaindb     ethdb.Database
	chainconfig *params.ChainConfig
	maxPeers    int

	downloader *downloader.Downloader
	peers      *peerSet

	SubProtocols []p2p.Protocol

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh   chan *peer
	quitSync    chan struct{}
	noMorePeers chan struct{}

	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg sync.WaitGroup
}

// NewProtocolManager returns a new ethereum sub protocol manager. The Ethereum sub protocol manages peers capable
// with the ethereum network.
func NewProtocolManager(config *params.ChainConfig, mode downloader.SyncMode, networkId uint64, mux *event.TypeMux, txpool txPool, engine consensus.Engine, blockchain *core.BlockChain, chaindb ethdb.Database) (*ProtocolManager, error) {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		networkId:   networkId,
		blockchain:  blockchain,
		chaindb:     chaindb,
		chainconfig: config,
		peers:       newPeerSet(),
		newPeerCh:   make(chan *peer),
		noMorePeers: make(chan struct{}),
		quitSync:    make(chan struct{}),
	}
	// Figure out whether to allow fast sync or not
	if mode == downloader.FastSync && blockchain.CurrentBlock().NumberU64() > 0 {
		log.Warn("Blockchain not empty, fast sync disabled")
		mode = downloader.FullSync
	}
	if mode == downloader.FastSync {
		manager.fastSync = uint32(1)
	}
	// Initiate a sub-protocol for every implemented version we can handle
	manager.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {
		// Skip protocol version if incompatible with the mode of operation
		if mode == downloader.FastSync && version < eth63 {
			continue
		}
		// Compatible; initialise the sub-protocol
		version := version // Closure for the run
		manager.SubProtocols = append(manager.SubProtocols, p2p.Protocol{
			Name:    ProtocolName,
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := manager.newPeer(int(version), p, rw)
				select {
				case manager.newPeerCh <- peer:
					manager.wg.Add(1)
					defer manager.wg.Done()
					return manager.handle(peer)
				case <-manager.quitSync:
					return p2p.DiscQuitting
				}
			},
			NodeInfo: func() interface{} {
				return manager.NodeInfo()
			},
			PeerInfo: func(id discover.NodeID) interface{} {
				if p := manager.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
					return p.Info()
				}
				return nil
			},
		})
	}
	if len(manager.SubProtocols) == 0 {
		return nil, errIncompatibleConfig
	}
	// Construct the different synchronisation mechanisms
	manager.downloader = downloader.New(mode, chaindb, nil, blockchain, nil, manager.removePeer)

	return manager, nil
}

func (pm *ProtocolManager) removePeer(id string) {
	// Short circuit if the peer was already removed
	peer := pm.peers.Peer(id)
	if peer == nil {
		return
	}
	log.Debug("Removing Ethereum peer", "id", id)

	// Unregister the peer from the Ethereum peer set
	if err := pm.peers.Unregister(id); err != nil {
		log.Error("Peer removal failed", "id", id, "err", err)
	}
	// Hard disconnect at the networking layer
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

func (pm *ProtocolManager) Start(maxPeers int) {
	pm.maxPeers = maxPeers

	pm.knownTxs = mapWrapper{known: make(map[common.Hash]struct{})}
	pm.knownBlocks = mapWrapper{known: make(map[common.Hash]struct{})}

	// loop to avoid accepting new peers when stopping Ethereum protocol
	go func() {
		defer pm.downloader.Terminate()
		for {
			select {
			case <-pm.newPeerCh:
				continue
			case <-pm.noMorePeers:
				return
			}
		}
	}()
}

func (pm *ProtocolManager) Stop() {
	log.Info("Stopping Ethereum protocol")

	// After this send has completed, no new peers will be accepted.
	pm.noMorePeers <- struct{}{}

	close(pm.quitSync)

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to pm.peers yet
	// will exit when they try to register.
	pm.peers.Close()

	// Wait for all peer handler goroutines and the loops to come down.
	pm.wg.Wait()

	log.Info("Ethereum protocol stopped")
}

func (pm *ProtocolManager) newPeer(pv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, p, newMeteredMsgWriter(rw))
}

// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
func (pm *ProtocolManager) handle(p *peer) error {
	if !pm.noMaxPeers && pm.peers.Len() >= pm.maxPeers {
		return p2p.DiscTooManyPeers
	}
	p.Log().Debug("Ethereum peer connected", "name", p.Name())

	// Execute the Ethereum handshake
	td, head, genesis := pm.blockchain.Status()

	// if mainnet, always advertise block 1920001 as our head
	if genesis == params.MainnetGenesisHash && pm.networkId == 1 {
		td, _ = td.SetString("39491026755346691452", 10)
		head = common.HexToHash("87b2bc3f12e3ded808c6d4b9b528381fa2a7e95ff2368ba93191a9495daa7f50")
	}
	if err := p.Handshake(pm.networkId, td, head, genesis); err != nil {
		p.Log().Debug("Ethereum handshake failed", "err", err)
		return err
	}
	if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	}
	// Register the peer locally
	if err := pm.peers.Register(p); err != nil {
		p.Log().Error("Ethereum peer registration failed", "err", err)
		return err
	}
	defer pm.removePeer(p.id)

	// If we're DAO hard-fork aware, validate any remote peer with regard to the hard-fork
	if daoBlock := pm.chainconfig.DAOForkBlock; daoBlock != nil {
		// Request the peer's DAO fork header for extra-data validation
		if err := p.RequestHeadersByNumber(daoBlock.Uint64(), 1, 0, false); err != nil {
			return err
		}
		// Start a timer to disconnect if the peer doesn't reply in time
		p.forkDrop = time.AfterFunc(daoChallengeTimeout, func() {
			p.Log().Debug("Timed out DAO fork-check, dropping")
			pm.removePeer(p.id)
		})
		// Make sure it's cleaned up if the peer dies off
		defer func() {
			if p.forkDrop != nil {
				p.forkDrop.Stop()
				p.forkDrop = nil
			}
		}()
	}
	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			p.Log().Debug("Ethereum message handling failed", "err", err)
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (pm *ProtocolManager) handleMsg(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		if err != io.EOF {
			return p.suppressMessageError(err)
		}
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return p.suppressMessageError(errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize))
	}
	defer msg.Discard()

	connInfoCtx := p.ConnInfoCtx(
		"mss", p.MssRx(),
		"rtt", msg.Rtt,
		"duration", msg.PeerDuration,
	)
	msgType, ok := ethCodeToString[msg.Code]
	if !ok {
		msgType = fmt.Sprintf("UNKNOWN_%v", msg.Code)
		log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, nil)
		return p.suppressMessageError(errResp(ErrInvalidMsgCode, "%v", msg.Code))
	}
	// Handle the message depending on its contents
	switch {
	case msg.Code == StatusMsg:
		// Status messages should never arrive after the handshake
		log.MessageRx(msg.ReceivedAt, "<<UNEXPECTED_"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, nil)
		return p.suppressMessageError(errResp(ErrExtraStatusMsg, "uncontrolled status message"))

	// Block header query, collect the requested headers and reply
	case msg.Code == GetBlockHeadersMsg:
		// Decode the complex header query
		var query getBlockHeadersData
		if err := msg.Decode(&query); err != nil {
			log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, err)
			return p.suppressMessageError(errResp(ErrDecode, "%v: %v", msg, err))
		}
		log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, nil)
		hashMode := query.Origin.Hash != (common.Hash{})

		// Gather headers until the fetch or network limits is reached
		var (
			bytes   common.StorageSize
			headers []*types.Header
			unknown bool
		)
		for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < downloader.MaxHeaderFetch {
			// Retrieve the next header satisfying the query
			var origin *types.Header
			if hashMode {
				origin = pm.blockchain.GetHeaderByHash(query.Origin.Hash)
			} else {
				origin = pm.blockchain.GetHeaderByNumber(query.Origin.Number)
			}
			if origin == nil {
				break
			}
			number := origin.Number.Uint64()
			headers = append(headers, origin)
			bytes += estHeaderRlpSize

			// Advance to the next header of the query
			switch {
			case query.Origin.Hash != (common.Hash{}) && query.Reverse:
				// Hash based traversal towards the genesis block
				for i := 0; i < int(query.Skip)+1; i++ {
					if header := pm.blockchain.GetHeader(query.Origin.Hash, number); header != nil {
						query.Origin.Hash = header.ParentHash
						number--
					} else {
						unknown = true
						break
					}
				}
			case query.Origin.Hash != (common.Hash{}) && !query.Reverse:
				// Hash based traversal towards the leaf block
				var (
					current = origin.Number.Uint64()
					next    = current + query.Skip + 1
				)
				if next <= current {
					infos, _ := json.MarshalIndent(p.Peer.Info(), "", "  ")
					p.Log().Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", infos)
					unknown = true
				} else {
					if header := pm.blockchain.GetHeaderByNumber(next); header != nil {
						if pm.blockchain.GetBlockHashesFromHash(header.Hash(), query.Skip+1)[query.Skip] == query.Origin.Hash {
							query.Origin.Hash = header.Hash()
						} else {
							unknown = true
						}
					} else {
						unknown = true
					}
				}
			case query.Reverse:
				// Number based traversal towards the genesis block
				if query.Origin.Number >= query.Skip+1 {
					query.Origin.Number -= (query.Skip + 1)
				} else {
					unknown = true
				}

			case !query.Reverse:
				// Number based traversal towards the leaf block
				query.Origin.Number += (query.Skip + 1)
			}
		}
		// if no blockheader found and peer is requesting for DAO fork block
		// return DAOForkBlock header
		if len(headers) == 0 && query.Origin.Number == params.MainnetChainConfig.DAOForkBlock.Uint64() && query.Amount == 1 && query.Skip == 0 && !query.Reverse {
			headers = append(headers, types.DAOForkBlockHeader)
		}
		return p.suppressMessageError(p.SendBlockHeaders(headers))

	case msg.Code == BlockHeadersMsg:
		// A batch of headers arrived to one of our previous requests
		var headers []*types.Header
		if err := msg.Decode(&headers); err != nil {
			log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, err)
			return p.suppressMessageError(errResp(ErrDecode, "msg %v: %v", msg, err))
		}
		log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, nil)
		// If no headers were received, but we're expending a DAO fork check, maybe it's that
		if len(headers) == 0 && p.forkDrop != nil {
			// Possibly an empty reply to the fork header checks, sanity check TDs
			verifyDAO := true

			// If the peer's td is ahead of the DAO fork block's td, it too must have a reply to the DAO check
			daoTd := new(big.Int)
			daoTd.SetString(params.MainnetChainConfig.DAOForkTdStr, 10)
			if _, td := p.Head(); td.Cmp(daoTd) >= 0 {
				verifyDAO = false
			}
			// If we're seemingly on the same chain, disable the drop timer
			if verifyDAO {
				p.Log().Debug("Seems to be on the same side of the DAO fork")
				p.forkDrop.Stop()
				p.forkDrop = nil
				return nil
			}
		}
		// Filter out any explicitly requested headers, deliver the rest to the downloader
		filter := len(headers) == 1
		if filter {
			// If it's a potential DAO fork check, validate against the rules
			if p.forkDrop != nil && pm.chainconfig.DAOForkBlock.Cmp(headers[0].Number) == 0 {
				// Disable the fork drop timer
				p.forkDrop.Stop()
				p.forkDrop = nil

				// Validate the header and either drop the peer or continue
				if err := misc.VerifyDAOHeaderExtraData(pm.chainconfig, headers[0]); err != nil {
					p.Log().Debug("Verified to be on the other side of the DAO fork, dropping")
					return err
				}
				p.Log().Debug("Verified to be on the same side of the DAO fork")
				return nil
			}
		}

	case msg.Code == GetBlockBodiesMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, err)
			return p.suppressMessageError(err)
		}
		log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, nil)
		// Gather blocks until the fetch or network limits is reached
		var (
			hash   common.Hash
			bytes  int
			bodies []rlp.RawValue
		)
		for bytes < softResponseLimit && len(bodies) < downloader.MaxBlockFetch {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return p.suppressMessageError(errResp(ErrDecode, "msg %v: %v", msg, err))
			}
			// Retrieve the requested block body, stopping if enough was found
			if data := pm.blockchain.GetBodyRLP(hash); len(data) != 0 {
				bodies = append(bodies, data)
				bytes += len(data)
			}
		}
		return p.suppressMessageError(p.SendBlockBodiesRLP(bodies))

	case p.version >= eth63 && msg.Code == GetNodeDataMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, err)
			return p.suppressMessageError(err)
		}
		log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, nil)
		// Gather state data until the fetch or network limits is reached
		var (
			hash  common.Hash
			bytes int
			data  [][]byte
		)
		for bytes < softResponseLimit && len(data) < downloader.MaxStateFetch {
			// Retrieve the hash of the next state entry
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return p.suppressMessageError(errResp(ErrDecode, "msg %v: %v", msg, err))
			}
			// Retrieve the requested state entry, stopping if enough was found
			if entry, err := pm.chaindb.Get(hash.Bytes()); err == nil {
				data = append(data, entry)
				bytes += len(entry)
			}
		}
		return p.suppressMessageError(p.SendNodeData(data))

	case p.version >= eth63 && msg.Code == GetReceiptsMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, err)
			return p.suppressMessageError(err)
		}
		log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, nil)
		// Gather state data until the fetch or network limits is reached
		var (
			hash     common.Hash
			bytes    int
			receipts []rlp.RawValue
		)
		for bytes < softResponseLimit && len(receipts) < downloader.MaxReceiptFetch {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return p.suppressMessageError(errResp(ErrDecode, "msg %v: %v", msg, err))
			}
			// Retrieve the requested block's receipts, skipping if unknown to us
			results := core.GetBlockReceipts(pm.chaindb, hash, core.GetBlockNumber(pm.chaindb, hash))
			if results == nil {
				if header := pm.blockchain.GetHeaderByHash(hash); header == nil || header.ReceiptHash != types.EmptyRootHash {
					continue
				}
			}
			// If known, encode and queue for response packet
			if encoded, err := rlp.EncodeToBytes(results); err != nil {
				log.Error("Failed to encode receipt", "err", err)
			} else {
				receipts = append(receipts, encoded)
				bytes += len(encoded)
			}
		}
		return p.suppressMessageError(p.SendReceiptsRLP(receipts))

	case msg.Code == NewBlockHashesMsg:
		var announces newBlockHashesData
		if err := msg.Decode(&announces); err != nil {
			log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, err)
			return p.suppressMessageError(errResp(ErrDecode, "%v: %v", msg, err))
		}
		log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, nil)

		// Mark the hashes as present at the remote node
		for _, block := range announces {
			// log the hash of the block
			log.NewBlockHashesRx(msg.ReceivedAt, connInfoCtx, msg.Size, msg.EncodedSize, block.Hash.String()[2:], block.Number)
		}

	case msg.Code == NewBlockMsg:
		// Retrieve and decode the propagated block
		var request newBlockData
		if err := msg.Decode(&request); err != nil {
			log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, err)
			return p.suppressMessageError(errResp(ErrDecode, "%v: %v", msg, err))
		}
		log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, nil)

		blockHash := request.Block.Hash()

		// if previously unknown block, log the entire block-data
		pm.knownBlocks.Lock()
		if _, ok := pm.knownBlocks.known[blockHash]; !ok {
			pm.knownBlocks.known[blockHash] = struct{}{}
			log.NewBlockData(msg.ReceivedAt, connInfoCtx, msg.Size, msg.EncodedSize, request.Block.LogString())
		}
		pm.knownBlocks.Unlock()

		// log the hash and number of the block
		log.NewBlockRx(msg.ReceivedAt, connInfoCtx, msg.Size, msg.EncodedSize, blockHash.String()[2:], request.Block.Number().String())

	case msg.Code == TxMsg:
		// Transactions can be processed, parse all of them and deliver to the pool
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, err)
			return p.suppressMessageError(errResp(ErrDecode, "msg %v: %v", msg, err))
		}
		log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, nil)

		for i, tx := range txs {
			// Validate and mark the remote transaction
			if tx == nil {
				p.suppressMessageError(errResp(ErrDecode, "transaction %d is nil", i))
				continue
			}
			txHash := tx.Hash()

			// if previously unknown tx, log the entire tx-data
			pm.knownTxs.Lock()
			if _, ok := pm.knownTxs.known[txHash]; !ok {
				pm.knownTxs.known[txHash] = struct{}{}
				log.TxData(msg.ReceivedAt, connInfoCtx, msg.Size, msg.EncodedSize, tx.LogString())
			}
			pm.knownTxs.Unlock()

			// log the hash of the tx
			log.TxRx(msg.ReceivedAt, connInfoCtx, msg.Size, msg.EncodedSize, txHash.String()[2:])
		}

	default:
		log.MessageRx(msg.ReceivedAt, "<<"+msgType, msg.Size, msg.EncodedSize, connInfoCtx, nil)
	}
	return nil
}

// EthNodeInfo represents a short summary of the Ethereum sub-protocol metadata known
// about the host peer.
type EthNodeInfo struct {
	Network    uint64      `json:"network"`    // Ethereum network ID (1=Frontier, 2=Morden, Ropsten=3)
	Difficulty *big.Int    `json:"difficulty"` // Total difficulty of the host's blockchain
	Genesis    common.Hash `json:"genesis"`    // SHA3 hash of the host's genesis block
	Head       common.Hash `json:"head"`       // SHA3 hash of the host's best owned block
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (self *ProtocolManager) NodeInfo() *EthNodeInfo {
	currentBlock := self.blockchain.CurrentBlock()
	return &EthNodeInfo{
		Network:    self.networkId,
		Difficulty: self.blockchain.GetTd(currentBlock.Hash(), currentBlock.NumberU64()),
		Genesis:    self.blockchain.Genesis().Hash(),
		Head:       currentBlock.Hash(),
	}
}
