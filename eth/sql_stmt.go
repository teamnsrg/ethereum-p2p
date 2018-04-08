package eth

import (
	"github.com/teamnsrg/go-ethereum/p2p"
	"github.com/teamnsrg/go-ethereum/p2p/discover"
)

func (pm *ProtocolManager) queueNodeEthInfo(id discover.NodeID, newInfo *p2p.Info, newStatus bool) {
	pm.ethInfoChan <- []interface{}{
		id.String(), newInfo.IP, newInfo.TCPPort, newInfo.RemotePort,
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
