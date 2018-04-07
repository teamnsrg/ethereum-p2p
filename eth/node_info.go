package eth

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/teamnsrg/go-ethereum/crypto"
	"github.com/teamnsrg/go-ethereum/log"
	"github.com/teamnsrg/go-ethereum/p2p"
)

func (pm *ProtocolManager) storeNodeEthInfo(p *peer, statusWrapper *statusDataWrapper) {
	connInfoCtx := p.ConnInfoCtx()
	id := p.ID()
	status := statusWrapper.Status
	receivedTd := p2p.NewTd(status.TD)
	receivedAt := &p2p.UnixTime{Time: statusWrapper.ReceivedAt}

	newInfo := &p2p.Info{
		ProtocolVersion: uint64(status.ProtocolVersion),
		NetworkId:       status.NetworkId,
		FirstReceivedTd: receivedTd,
		LastReceivedTd:  receivedTd,
		BestHash:        status.CurrentBlock.String()[2:],
		GenesisHash:     status.GenesisBlock.String()[2:],
		FirstStatusAt:   receivedAt,
		LastStatusAt:    receivedAt,
	}

	currentInfo := pm.knownNodeInfos.GetInfo(id)
	if currentInfo == nil {
		if err := pm.fillP2PInfo(p, newInfo); err != nil {
			p.Log().Debug("Failed to fill P2P info", "err", err)
			log.Status(*newInfo.LastStatusAt.Time, connInfoCtx, statusWrapper.PeerRtt, statusWrapper.PeerDuration, newInfo.Hello(), newInfo.Status())
			return
		}
		newInfo.RLock()
		defer newInfo.RUnlock()
		// add new node info to in-memory
		pm.knownNodeInfos.AddInfo(id, newInfo)
	} else {
		currentInfo.Lock()
		defer currentInfo.Unlock()
		currentInfo.LastStatusAt = newInfo.LastStatusAt
		currentInfo.LastReceivedTd = newInfo.LastReceivedTd
		currentInfo.BestHash = newInfo.BestHash
		if currentInfo.FirstStatusAt == nil || isNewEthNode(currentInfo, newInfo) {
			currentInfo.FirstStatusAt = newInfo.FirstStatusAt
			currentInfo.FirstReceivedTd = newInfo.FirstReceivedTd
			currentInfo.ProtocolVersion = newInfo.ProtocolVersion
			currentInfo.NetworkId = newInfo.NetworkId
			currentInfo.GenesisHash = newInfo.GenesisHash
		}
		newInfo = currentInfo
	}

	log.Status(*newInfo.LastStatusAt.Time, connInfoCtx, statusWrapper.PeerRtt, statusWrapper.PeerDuration, newInfo.Hello(), newInfo.Status())

	// queue updated/new entry to node_eth_info
	if pm.ethInfoChan != nil {
		log.Sql("Queueing NodeEthInfo", connInfoCtx...)
		pm.queueNodeEthInfo(id, newInfo, true)
	}
}

func (pm *ProtocolManager) fillP2PInfo(p *peer, newInfo *p2p.Info) error {
	id := p.ID()
	newInfo.Keccak256Hash = crypto.Keccak256Hash(id[:]).String()[2:]
	remoteAddr, ok := p.RemoteAddr().(*net.TCPAddr)
	if ok {
		newInfo.IP = remoteAddr.IP.String()
		newInfo.RemotePort = uint16(remoteAddr.Port)
	}
	newInfo.TCPPort = p.TCPPort()

	newInfo.ListenPort = p.ListenPort()
	newInfo.P2PVersion = p.Version()

	clientId := p.Name()
	var capsArray []string
	for _, c := range p.Caps() {
		capsArray = append(capsArray, c.String())
	}
	sort.Strings(capsArray)
	caps := strings.Join(capsArray, ",")

	// replace unwanted characters
	if pm.strReplacer == nil {
		return fmt.Errorf("pm.strReplacer == nil")
	}
	newInfo.ClientId = pm.strReplacer.Replace(clientId)
	newInfo.Caps = pm.strReplacer.Replace(caps)
	return nil
}

func isNewEthNode(oldInfo *p2p.Info, newInfo *p2p.Info) bool {
	return oldInfo.ProtocolVersion != newInfo.ProtocolVersion || oldInfo.NetworkId != newInfo.NetworkId ||
		oldInfo.GenesisHash != newInfo.GenesisHash
}

func (pm *ProtocolManager) storeDAOForkSupportInfo(p *peer, msg *p2p.Msg, daoForkSupport int8) {
	connInfoCtx := p.ConnInfoCtx()
	id := p.ID()

	currentInfo := pm.knownNodeInfos.GetInfo(id)
	// missing other node information
	// log but don't update database
	if currentInfo == nil {
		log.DaoFork(msg.ReceivedAt, connInfoCtx, msg.PeerRtt, msg.PeerDuration, daoForkSupport > 0)
		return
	}

	currentInfo.Lock()
	defer currentInfo.Unlock()
	// daoForkSupport hasn't changed
	// log but don't update database
	if currentInfo.DAOForkSupport == daoForkSupport {
		log.DaoFork(msg.ReceivedAt, connInfoCtx, msg.PeerRtt, msg.PeerDuration, daoForkSupport > 0)
		return
	}
	currentInfo.DAOForkSupport = daoForkSupport

	log.DaoFork(msg.ReceivedAt, connInfoCtx, msg.PeerRtt, msg.PeerDuration, daoForkSupport > 0)

	// queue updated/new entry to node_eth_info
	if pm.ethInfoChan != nil {
		log.Sql("Queueing NodeEthInfo", connInfoCtx...)
		pm.queueNodeEthInfo(id, currentInfo, false)
	}
}
