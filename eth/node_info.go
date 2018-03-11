package eth

import (
	"time"

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

	var infoStr string
	if currentInfo, ok := pm.knownNodeInfos[id]; !ok {
		infoStr = newInfo.EthSummary()
	} else {
		currentInfo.Lock()
		defer currentInfo.Unlock()
		currentInfo.LastStatusAt = newInfo.LastStatusAt
		currentInfo.LastReceivedTd = newInfo.LastReceivedTd
		currentInfo.BestHash = newInfo.BestHash
		if currentInfo.FirstStatusAt == nil {
			// add new eth info to existing entry
			currentInfo.FirstStatusAt = newInfo.FirstStatusAt
			currentInfo.FirstReceivedTd = newInfo.FirstReceivedTd
			currentInfo.ProtocolVersion = newInfo.ProtocolVersion
			currentInfo.NetworkId = newInfo.NetworkId
			currentInfo.GenesisHash = newInfo.GenesisHash
			if pm.db != nil {
				pm.addEthInfo(&p2p.KnownNodeInfosWrapper{NodeId: nodeid, Info: currentInfo})
			}
		} else if isNewEthNode(currentInfo, newInfo) {
			// a new entry, including address and DEVp2p info, is added to mysql db
			currentInfo.FirstStatusAt = newInfo.FirstStatusAt
			currentInfo.FirstReceivedTd = newInfo.FirstReceivedTd
			currentInfo.ProtocolVersion = newInfo.ProtocolVersion
			currentInfo.NetworkId = newInfo.NetworkId
			currentInfo.GenesisHash = newInfo.GenesisHash
			if pm.db != nil {
				pm.addEthNodeInfo(&p2p.KnownNodeInfosWrapper{NodeId: nodeid, Info: currentInfo}, true)
				if rowId := pm.getRowID(nodeid); rowId > 0 {
					currentInfo.RowId = rowId
				}
			}
		} else {
			if pm.db != nil {
				// update eth info
				pm.updateEthInfo(&p2p.KnownNodeInfosWrapper{NodeId: nodeid, Info: currentInfo})
			}
		}
		infoStr = currentInfo.EthSummary()
	}
	log.Info("[STATUS]", "receivedAt", receivedAt, "id", nodeid, "conn", p.ConnFlags(), "info", infoStr)

}

func isNewEthNode(oldInfo *p2p.Info, newInfo *p2p.Info) bool {
	return oldInfo.ProtocolVersion != newInfo.ProtocolVersion || oldInfo.NetworkId != newInfo.NetworkId ||
		oldInfo.GenesisHash != newInfo.GenesisHash
}

func (pm *ProtocolManager) storeDAOForkSupportInfo(p *peer, receivedAt time.Time, daoForkSupport int8) {
	id := p.ID()
	nodeid := id.String()

	if currentInfo, ok := pm.knownNodeInfos[id]; ok {
		currentInfo.Lock()
		defer currentInfo.Unlock()
		if currentInfo.DAOForkSupport == 0 {
			// add DAOForkSupport flag to existing entry for the first time
			currentInfo.DAOForkSupport = daoForkSupport
			pm.addDAOForkSupport(&p2p.KnownNodeInfosWrapper{NodeId: nodeid, Info: currentInfo})
		} else if currentInfo.DAOForkSupport != daoForkSupport {
			// DAOForkSupport flag value changed. add a new entry to mysql db
			currentInfo.DAOForkSupport = daoForkSupport
			pm.addEthNodeInfo(&p2p.KnownNodeInfosWrapper{NodeId: nodeid, Info: currentInfo}, false)
			if rowId := pm.getRowID(nodeid); rowId > 0 {
				currentInfo.RowId = rowId
			}
		}
	}
	log.Info("[DAOFORK]", "receivedAt", receivedAt, "id", nodeid, "conn", p.ConnFlags(), "support", daoForkSupport > 0)
}
