package eth

import (
	"fmt"
	"strings"

	"github.com/teamnsrg/go-ethereum/log"
	"github.com/teamnsrg/go-ethereum/p2p"
)

func (pm *ProtocolManager) prepareSqlStmts() error {
	if pm.db != nil {
		if err := pm.prepareAddEthInfoStmt(); err != nil {
			return err
		}
		if err := pm.prepareUpdateEthInfoStmt(); err != nil {
			return err
		}
		if err := pm.prepareAddEthNodeInfoStmt(); err != nil {
			return err
		}
		if err := pm.prepareAddDAOForkSupportStmt(); err != nil {
			return err
		}
	}
	return nil
}

func (pm *ProtocolManager) closeSqlStmts() {
	if pm.addEthInfoStmt != nil {
		if err := pm.addEthInfoStmt.Close(); err != nil {
			log.Error("Failed to close AddEthInfo sql statement", "err", err)
		} else {
			log.Trace("Closed AddEthInfo sql statement")
		}
	}
	if pm.updateEthInfoStmt != nil {
		if err := pm.updateEthInfoStmt.Close(); err != nil {
			log.Error("Failed to close UpdateEthInfo sql statement", "err", err)
		} else {
			log.Trace("Closed UpdateEthInfo sql statement")
		}
	}
	if pm.addEthNodeInfoStmt != nil {
		if err := pm.addEthNodeInfoStmt.Close(); err != nil {
			log.Error("Failed to close AddEthNodeInfo sql statement", "err", err)
		} else {
			log.Trace("Closed AddEthNodeInfo sql statement")
		}
	}
	if pm.addDAOForkSupportStmt != nil {
		if err := pm.addDAOForkSupportStmt.Close(); err != nil {
			log.Error("Failed to close AddDAOForkSupport sql statement", "err", err)
		} else {
			log.Trace("Closed AddDAOForkSupport sql statement")
		}
	}
}

func (pm *ProtocolManager) prepareAddEthInfoStmt() error {
	pStmt, err := pm.db.Prepare("UPDATE node_info " +
		"SET protocol_version=?, network_id=?, first_received_td=?, last_received_td=?, " +
		"best_hash=?, genesis_hash=?, first_status_at=?, last_status_at=? WHERE id=?")
	if err != nil {
		log.Error("Failed to prepare AddEthInfo sql statement", "err", err)
		return err
	} else {
		log.Trace("Prepared AddEthInfo sql statement")
		pm.addEthInfoStmt = pStmt
	}
	return nil
}

func (pm *ProtocolManager) addEthInfo(newInfoWrapper *p2p.KnownNodeInfosWrapper) {
	// exit if no prepared statement
	if pm.addEthInfoStmt == nil {
		log.Crit("No prepared statement for AddEthInfo")
		return
	}

	nodeid := newInfoWrapper.NodeId
	newInfo := newInfoWrapper.Info
	lastStatusAt := newInfo.LastStatusAt.Float64()
	lastReceivedTd := newInfo.LastReceivedTd.String()
	_, err := pm.addEthInfoStmt.Exec(newInfo.ProtocolVersion, newInfo.NetworkId, lastReceivedTd, lastReceivedTd,
		newInfo.BestHash, newInfo.GenesisHash, lastStatusAt, lastStatusAt, newInfo.RowID)
	if err != nil {
		log.Error("Failed to execute AddEthInfo sql statement", "id", nodeid, "newInfo", newInfo, "err", err)
	} else {
		log.Debug("Executed AddEthInfo sql statement", "id", nodeid, "newInfo", newInfo)
	}
}

func (pm *ProtocolManager) prepareUpdateEthInfoStmt() error {
	pStmt, err := pm.db.Prepare("UPDATE node_info SET last_received_td=?, best_hash=?, last_status_at=? WHERE id=?")

	if err != nil {
		log.Error("Failed to prepare UpdateEthInfo sql statement", "err", err)
		return err
	} else {
		log.Trace("Prepared UpdateEthInfo sql statement")
		pm.updateEthInfoStmt = pStmt
	}
	return nil
}

func (pm *ProtocolManager) updateEthInfo(newInfoWrapper *p2p.KnownNodeInfosWrapper) {
	// exit if no prepared statement
	if pm.updateEthInfoStmt == nil {
		log.Crit("No prepared statement for UpdateEthInfo")
		return
	}

	nodeid := newInfoWrapper.NodeId
	newInfo := newInfoWrapper.Info
	lastStatusAt := newInfo.LastStatusAt.Float64()
	lastReceivedTd := newInfo.LastReceivedTd.String()
	_, err := pm.updateEthInfoStmt.Exec(lastReceivedTd, newInfo.BestHash, lastStatusAt, newInfo.RowID)
	if err != nil {
		log.Error("Failed to execute UpdateEthInfo sql statement", "id", nodeid, "newInfo", newInfo, "err", err)
	} else {
		log.Debug("Executed UpdateEthInfo sql statement", "id", nodeid, "newInfo", newInfo)
	}
}

func (pm *ProtocolManager) prepareAddEthNodeInfoStmt() error {
	fields := []string{"node_id", "ip", "tcp_port", "remote_port", "p2p_version", "client_id", "caps", "listen_port",
		"first_hello_at", "last_hello_at", "protocol_version", "network_id", "first_received_td", "last_received_td",
		"best_hash", "genesis_hash", "dao_fork", "first_status_at", "last_status_at"}

	stmt := fmt.Sprintf(`INSERT INTO node_info (%s) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.Join(fields, ", "))
	pStmt, err := pm.db.Prepare(stmt)
	if err != nil {
		log.Error("Failed to prepare AddEthNodeInfo sql statement", "err", err)
		return err
	} else {
		log.Trace("Prepared AddEthNodeInfo sql statement")
		pm.addEthNodeInfoStmt = pStmt
	}
	return nil
}

func (pm *ProtocolManager) addEthNodeInfo(newInfoWrapper *p2p.KnownNodeInfosWrapper) {
	// exit if no prepared statement
	if pm.addEthNodeInfoStmt == nil {
		log.Crit("No prepared statement for AddEthNodeInfo")
		return
	}

	nodeid := newInfoWrapper.NodeId
	newInfo := newInfoWrapper.Info
	firstHelloAt := newInfo.FirstHelloAt.Float64()
	lastHelloAt := newInfo.LastHelloAt.Float64()
	firstStatusAt := newInfo.FirstStatusAt.Float64()
	lastStatusAt := newInfo.LastStatusAt.Float64()
	firstReceivedTd := newInfo.FirstReceivedTd.String()
	lastReceivedTd := newInfo.LastReceivedTd.String()
	_, err := pm.addEthNodeInfoStmt.Exec(nodeid, newInfo.IP, newInfo.TCPPort, newInfo.RemotePort,
		newInfo.P2PVersion, newInfo.ClientId, newInfo.Caps, newInfo.ListenPort, firstHelloAt, lastHelloAt,
		newInfo.ProtocolVersion, newInfo.NetworkId, firstReceivedTd, lastReceivedTd, newInfo.BestHash, newInfo.GenesisHash,
		newInfo.DAOForkSupport, firstStatusAt, lastStatusAt)
	if err != nil {
		log.Error("Failed to execute AddEthNodeInfo sql statement", "id", nodeid, "newInfo", newInfo, "err", err)
	} else {
		log.Debug("Executed AddEthNodeInfo sql statement", "id", nodeid, "newInfo", newInfo)
	}
}

func (pm *ProtocolManager) prepareAddDAOForkSupportStmt() error {
	pStmt, err := pm.db.Prepare("UPDATE node_info SET dao_fork=? WHERE id=?")
	if err != nil {
		log.Error("Failed to prepare AddDAOForkSupport sql statement", "err", err)
		return err
	} else {
		log.Trace("Prepared AddDAOForkSupport sql statement")
		pm.addDAOForkSupportStmt = pStmt
	}
	return nil
}

func (pm *ProtocolManager) addDAOForkSupport(newInfoWrapper *p2p.KnownNodeInfosWrapper) {
	// exit if no prepared statement
	if pm.addDAOForkSupportStmt == nil {
		log.Crit("No prepared statement for AddDAOForkSupport")
		return
	}

	newInfo := newInfoWrapper.Info
	nodeid := newInfoWrapper.NodeId
	_, err := pm.addDAOForkSupportStmt.Exec(newInfo.DAOForkSupport, newInfo.RowID)
	if err != nil {
		log.Error("Failed to execute AddDAOForkSupport sql statement", "id", nodeid, "daoForkSupport", newInfo.DAOForkSupport, "err", err)
	} else {
		log.Debug("Executed AddDAOForkSupport sql statement", "id", nodeid, "daoForkSupport", newInfo.DAOForkSupport)
	}
}

func (pm *ProtocolManager) getRowID(nodeid string) uint64 {
	// exit if no prepared statement
	if pm.getRowIDStmt == nil {
		log.Crit("No prepared statement for AddEthNodeInfo")
		return 0
	}

	var rowID uint64
	err := pm.getRowIDStmt.QueryRow(nodeid).Scan(&rowID)
	if err != nil {
		log.Error("Failed to execute GetRowID sql statement", "id", nodeid, "err", err)
		return 0
	} else {
		log.Debug("Executed GetRowID sql statement", "id", nodeid, "rowid", rowID)
		return rowID
	}
}
