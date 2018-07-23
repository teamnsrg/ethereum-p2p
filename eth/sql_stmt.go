package eth

import (
	"fmt"

	"github.com/teamnsrg/ethereum-p2p/p2p"
	"github.com/teamnsrg/ethereum-p2p/p2p/discover"
)

func (pm *ProtocolManager) queueNodeEthInfo(id discover.NodeID, newInfo *p2p.Info, newStatus bool) error {
	if newInfo.FirstReceivedTd == nil {
		return fmt.Errorf("FirstReceivedTd == nil")
	}
	if newInfo.LastReceivedTd == nil {
		return fmt.Errorf("LastReceivedTd == nil")
	}
	if newInfo.FirstStatusAt == nil {
		return fmt.Errorf("FirstStatusAt == nil")
	}
	if newInfo.LastStatusAt == nil {
		return fmt.Errorf("LastStatusAt == nil")
	}
	pm.ethInfoChan <- []interface{}{
		id.String(), newInfo.IP, newInfo.TCPPort, newInfo.RemotePort,
		newInfo.P2PVersion, newInfo.ClientId, newInfo.Caps, newInfo.ListenPort,
		newInfo.ProtocolVersion, newInfo.NetworkId, newInfo.FirstReceivedTd.String(), newInfo.LastReceivedTd.String(),
		newInfo.BestHash, newInfo.GenesisHash, newInfo.DAOForkSupport,
		newInfo.FirstStatusAt.Float64(), newInfo.LastStatusAt.Float64(), boolToInt(newStatus),
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
