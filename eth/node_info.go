package eth

import (
	"github.com/teamnsrg/go-ethereum/log"
	"github.com/teamnsrg/go-ethereum/p2p"
	"github.com/teamnsrg/go-ethereum/p2p/discover"
)

func (pm *ProtocolManager) storeEthNodeInfo(id discover.NodeID, statusWrapper *statusDataWrapper) {
	nodeid := id.String()
	status := statusWrapper.Status
	receivedAt := statusWrapper.ReceivedAt

	newInfo := &p2p.Info{
		ProtocolVersion: uint64(status.ProtocolVersion),
		NetworkId:       status.NetworkId,
		FirstReceivedTd: status.TD,
		LastReceivedTd:  status.TD,
		BestHash:        status.CurrentBlock.String()[2:],
		GenesisHash:     status.GenesisBlock.String()[2:],
	}

	if currentInfo, ok := pm.knownNodeInfos[id]; ok {
		currentInfo.Lock()
		defer currentInfo.Unlock()
		currentInfo.LastStatusAt = receivedAt
		currentInfo.LastReceivedTd = newInfo.LastReceivedTd
		currentInfo.BestHash = newInfo.BestHash
		if currentInfo.FirstStatusAt == nil {
			// add new eth info to existing entry
			currentInfo.FirstStatusAt = receivedAt
			currentInfo.FirstReceivedTd = newInfo.FirstReceivedTd
			currentInfo.ProtocolVersion = newInfo.ProtocolVersion
			currentInfo.NetworkId = newInfo.NetworkId
			currentInfo.GenesisHash = newInfo.GenesisHash
			if pm.addEthInfoStmt == nil {
				log.Crit("No prepared statement for AddEthInfo")
			}
			pm.addEthInfo(&p2p.KnownNodeInfosWrapper{nodeid, currentInfo})
		} else if isNewEthNode(currentInfo, newInfo) {
			// a new entry, including address and DEVp2p info, is added to mysql db
			currentInfo.ProtocolVersion = newInfo.ProtocolVersion
			currentInfo.NetworkId = newInfo.NetworkId
			currentInfo.GenesisHash = newInfo.GenesisHash
			if pm.addEthNodeInfoStmt == nil {
				log.Crit("No prepared statement for AddEthNodeInfo")
			}
			pm.addEthNodeInfo(&p2p.KnownNodeInfosWrapper{nodeid, currentInfo})
			if pm.getRowIDStmt == nil {
				log.Crit("No prepared statement for AddEthNodeInfo")
			}
			if rowID := pm.getRowID(nodeid); rowID > 0 {
				currentInfo.RowID = rowID
			}
		} else {
			// update eth info
			if pm.updateEthInfoStmt == nil {
				log.Crit("No prepared statement for UpdateEthInfo")
			}
			pm.updateEthInfo(&p2p.KnownNodeInfosWrapper{nodeid, currentInfo})
		}
	}
}

func isNewEthNode(oldInfo *p2p.Info, newInfo *p2p.Info) bool {
	return oldInfo.ProtocolVersion != newInfo.ProtocolVersion || oldInfo.NetworkId != newInfo.NetworkId ||
		oldInfo.GenesisHash != newInfo.GenesisHash
}
