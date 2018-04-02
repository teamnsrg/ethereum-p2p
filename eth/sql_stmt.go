package eth

import (
	"github.com/teamnsrg/go-ethereum/log"
	"github.com/teamnsrg/go-ethereum/p2p"
)

func (pm *ProtocolManager) prepareSqlStmts() error {
	if pm.db != nil {
		if err := pm.prepareAddNodeEthInfoStmt(); err != nil {
			return err
		}
	}
	return nil
}

func (pm *ProtocolManager) closeSqlStmts() {
	if pm.addNodeEthInfoStmt != nil {
		if err := pm.addNodeEthInfoStmt.Close(); err != nil {
			log.Error("Failed to close AddEthNodeInfo sql statement", "err", err)
		} else {
			log.Trace("Closed AddEthNodeInfo sql statement")
		}
	}
}

func (pm *ProtocolManager) prepareAddNodeEthInfoStmt() error {
	pStmt, err := pm.db.Prepare(`
		INSERT INTO node_eth_info 
			(node_id, ip, tcp_port, remote_port, 
			 p2p_version, client_id, caps, listen_port, 
			 protocol_version, network_id, first_received_td, last_received_td, 
			 best_hash, genesis_hash, dao_fork, first_status_at, last_status_at, status_count) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE 
			remote_port=VALUES(remote_port), 
			last_received_td=VALUES(last_received_td), 
			best_hash=VALUES(best_hash), 
			dao_fork=VALUES(dao_fork), 
			last_status_at=VALUES(last_status_at), 
			status_count=status_count+VALUES(status_count)
	`)
	if err != nil {
		log.Error("Failed to prepare AddNodeEthInfo sql statement", "err", err)
		return err
	} else {
		log.Trace("Prepared AddNodeEthInfo sql statement")
		pm.addNodeEthInfoStmt = pStmt
	}
	return nil
}

func (pm *ProtocolManager) addNodeEthInfo(newInfoWrapper *p2p.KnownNodeInfosWrapper, newStatus bool) {
	// exit if no prepared statement
	if pm.addNodeEthInfoStmt == nil {
		log.Crit("No prepared statement for AddNodeEthInfo")
		return
	}

	nodeid := newInfoWrapper.NodeId
	newInfo := newInfoWrapper.Info
	firstStatusAt := newInfo.FirstStatusAt.Float64()
	lastStatusAt := newInfo.LastStatusAt.Float64()
	firstReceivedTd := newInfo.FirstReceivedTd.String()
	lastReceivedTd := newInfo.LastReceivedTd.String()
	_, err := pm.addNodeEthInfoStmt.Exec(nodeid, newInfo.IP, newInfo.TCPPort, newInfo.RemotePort,
		newInfo.P2PVersion, newInfo.ClientId, newInfo.Caps, newInfo.ListenPort,
		newInfo.ProtocolVersion, newInfo.NetworkId, firstReceivedTd, lastReceivedTd, newInfo.BestHash, newInfo.GenesisHash,
		newInfo.DAOForkSupport, firstStatusAt, lastStatusAt, boolToInt(newStatus))
	if err != nil {
		log.Error("Failed to execute AddNodeEthInfo sql statement", "id", nodeid[:16], "err", err)
	} else {
		log.Trace("Executed AddNodeEthInfo sql statement", "id", nodeid[:16])
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
