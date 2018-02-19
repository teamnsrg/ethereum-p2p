package discover

import (
	"github.com/teamnsrg/go-ethereum/crypto"
	"github.com/teamnsrg/go-ethereum/log"
)

func (t *udp) prepareAddNeighborStmt() error {
	pStmt, err := t.sqldb.Prepare(`
		INSERT INTO neighbors (node_id, hash, ip, tcp_port, udp_port, first_received_at, last_received_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?) 
		ON DUPLICATE KEY UPDATE 
		last_received_at=VALUES(last_received_at), 
		count=count+1
		`)
	if err != nil {
		log.Error("Failed to prepare AddNeighbor sql statement", "err", err)
		return err
	} else {
		log.Trace("Prepared AddNeighbor sql statement")
		t.addNeighborStmt = pStmt
	}
	return nil
}

func (t *udp) addNeighbor(node rpcNode, unixTime float64) {
	// exit if no prepared statement
	if t.addNeighborStmt == nil {
		log.Crit("No prepared statement for AddNeighbor")
		return
	}
	nodeid := node.ID.String()
	hash := crypto.Keccak256Hash(node.ID[:]).String()[2:]
	ip := node.IP.String()
	tcpPort := node.TCP
	udpPort := node.UDP
	_, err := t.addNeighborStmt.Exec(nodeid, hash, ip, tcpPort, udpPort, unixTime, unixTime)
	if err != nil {
		log.Error("Failed to execute AddNeighbor sql statement", "node", node, "receivedAt", unixTime, "err", err)
	} else {
		log.Debug("Executed AddNeighbor sql statement", "node", node, "receivedAt", unixTime)
	}
}

func (t *udp) closeSqlStmts() {
	// close addNeighbor statement
	if t.addNeighborStmt != nil {
		if err := t.addNeighborStmt.Close(); err != nil {
			log.Error("Failed to close AddNeighbor sql statement", "err", err)
		} else {
			log.Trace("Closed AddNeighbor sql statement")
		}
	}
}
