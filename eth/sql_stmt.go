package eth

import (
	"github.com/teamnsrg/go-ethereum/log"
	"github.com/teamnsrg/go-ethereum/p2p"
)

func (pm *ProtocolManager) queueNodeEthInfo(newInfoWrapper *p2p.KnownNodeInfosWrapper, newStatus bool) {
	log.Sql("Queueing NodeEthInfo")
	newInfo := newInfoWrapper.Info
	pm.ethInfoChan <- []interface{}{
		newInfoWrapper.NodeId, newInfo.IP, newInfo.TCPPort, newInfo.RemotePort,
		newInfo.P2PVersion, newInfo.ClientId, newInfo.Caps, newInfo.ListenPort,
		newInfo.ProtocolVersion, newInfo.NetworkId, newInfo.FirstReceivedTd.String(), newInfo.LastReceivedTd.String(),
		newInfo.BestHash, newInfo.GenesisHash, newInfo.DAOForkSupport,
		newInfo.FirstStatusAt.Float64(), newInfo.LastStatusAt.Float64(), boolToInt(newStatus),
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
