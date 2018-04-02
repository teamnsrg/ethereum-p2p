package eth

import (
	"net"
	"sort"
	"strings"
	"time"

	"fmt"
	"github.com/teamnsrg/go-ethereum/crypto"
	"github.com/teamnsrg/go-ethereum/log"
	"github.com/teamnsrg/go-ethereum/p2p"
)

func (pm *ProtocolManager) storeEthNodeInfo(p *peer, statusWrapper *statusDataWrapper) {
	id := p.ID()
	nodeid := id.String()
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

	pm.knownNodeInfos.Lock()
	if currentInfo, ok := pm.knownNodeInfos.Infos()[id]; !ok {
		if err := pm.fillP2PInfo(p, newInfo); err != nil {
			pm.knownNodeInfos.Unlock()
			log.Debug("Failed to fill P2P info", "err", err)
			log.Info("[STATUS]", "receivedAt", receivedAt, "id", nodeid, "addr", p.RemoteAddr().String(), "conn", p.ConnFlags(), "info", newInfo.EthSummary())
			return
		}

		// add new node info to in-memory
		pm.knownNodeInfos.Infos()[id] = newInfo
	} else {
		currentInfo.Lock()
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
		currentInfo.Unlock()
		newInfo = currentInfo
	}
	pm.knownNodeInfos.Unlock()

	// update or add a new entry to node_eth_info
	if pm.db != nil {
		pm.addNodeEthInfo(&p2p.KnownNodeInfosWrapper{NodeId: nodeid, Info: newInfo}, true)
	}
	log.Info("[STATUS]", "receivedAt", receivedAt, "id", nodeid, "addr", p.RemoteAddr().String(), "conn", p.ConnFlags(), "info", newInfo.EthSummary())
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

func (pm *ProtocolManager) storeDAOForkSupportInfo(p *peer, receivedAt time.Time, daoForkSupport int8) {
	id := p.ID()
	nodeid := id.String()

	pm.knownNodeInfos.Lock()
	currentInfo, ok := pm.knownNodeInfos.Infos()[id]
	if ok {
		currentInfo.Lock()
		if currentInfo.DAOForkSupport == 0 || currentInfo.DAOForkSupport != daoForkSupport {
			currentInfo.DAOForkSupport = daoForkSupport
		}
		currentInfo.Unlock()
	}
	pm.knownNodeInfos.Unlock()

	// update or add a new entry to node_eth_info
	if pm.db != nil {
		pm.addNodeEthInfo(&p2p.KnownNodeInfosWrapper{NodeId: nodeid, Info: currentInfo}, false)
	}
	log.Info("[DAOFORK]", "receivedAt", receivedAt, "id", nodeid, "addr", p.RemoteAddr().String(), "conn", p.ConnFlags(), "support", daoForkSupport > 0)
}
