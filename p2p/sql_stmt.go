package p2p

import (
	"database/sql"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/teamnsrg/go-ethereum/log"
	"github.com/teamnsrg/go-ethereum/p2p/discover"
)

func (srv *Server) initSql() error {
	if srv.MySQLName == "" {
		log.Trace("No sql db connection info provided")
	} else {
		db, err := sql.Open("mysql", srv.MySQLName)
		if err != nil {
			log.Error("Failed to open sql db handle", "database", srv.MySQLName, "err", err)
			return err
		}
		log.Trace("Opened sql db handle", "database", srv.MySQLName)
		if err := db.Ping(); err != nil {
			log.Error("Sql db connection failed ping test", "database", srv.MySQLName, "err", err)
			return err
		}
		log.Trace("Sql db connection passed ping test", "database", srv.MySQLName)
		srv.DB = db

		// backup tables
		// If ResetSQL is true, BackupSQL should be true as well
		if srv.BackupSQL {
			sqlNameParsed := strings.Split(srv.MySQLName, "/")
			dbName := sqlNameParsed[len(sqlNameParsed)-1]
			for _, tableName := range []string{"neighbors", "node_meta_info", "node_info"} {
				if err := srv.backupTable(dbName, tableName); err != nil {
					return err
				}
			}
		}

		// reset tables
		if srv.ResetSQL {
			if err := srv.dropTables(); err != nil {
				return err
			}
		}

		if err := srv.createTables(); err != nil {
			return err
		}

		// fill KnownNodesInfos with info from the mysql database
		srv.loadKnownNodeInfos()

		// prepare sql statements
		if err := srv.prepareAddNodeInfoStmt(); err != nil {
			return err
		}
		if err := srv.prepareUpdateNodeInfoStmt(); err != nil {
			return err
		}
		if err := srv.prepareAddNodeMetaInfoStmt(); err != nil {
			return err
		}
		if err := srv.prepareGetRowID(); err != nil {
			return err
		}
	}
	return nil
}

func (srv *Server) checkIfTableExists(dbName string, tableName string) (int, error) {
	var result int
	err := srv.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM information_schema.TABLES 
		WHERE (TABLE_SCHEMA = ?) AND (TABLE_NAME = ?)
	`, dbName, tableName).Scan(&result)
	return result, err
}

func (srv *Server) backupTable(dbName string, tableName string) error {
	// check if table exists
	if result, err := srv.checkIfTableExists(dbName, tableName); err != nil {
		log.Error(fmt.Sprintf("Failed to check if %s table exists", tableName), "database", srv.MySQLName, "err", err)
		return err
	} else if result == 0 {
		log.Debug(tableName+" table not found. Skipping backup", "database", srv.MySQLName)
		return nil
	}
	log.Trace(tableName+" table exists", "database", srv.MySQLName)

	// format backup file name
	tableNameCamel := ""
	for _, s := range strings.Split(tableName, "_") {
		tableNameCamel += strings.Title(s)
	}
	currentTime := time.Now().UTC().Format("20060102T150405Z07")
	fileName := fmt.Sprintf("%s.sql-%s", tableNameCamel, currentTime)

	// create a table backup
	if _, err := srv.DB.Exec(fmt.Sprintf(`
		SELECT * INTO OUTFILE '/backup/%s' FROM %s
	`, fileName, tableName)); err != nil {
		log.Error(fmt.Sprintf("Failed to create %s", fileName), "database", srv.MySQLName, "err", err)
		return err
	}
	log.Trace(fileName+" created", "database", srv.MySQLName)
	return nil
}

func (srv *Server) dropTables() error {
	if _, err := srv.DB.Exec(`
		DROP TABLE IF EXISTS node_info, node_meta_info, neighbors
	`); err != nil {
		log.Error("Failed to drop sql tables", "database", srv.MySQLName, "err", err)
		return err
	}
	log.Trace("Existing sql tables dropped", "database", srv.MySQLName)
	return nil
}

func (srv *Server) createTables() error {
	// create neighbors table
	if _, err := srv.DB.Exec(`
		CREATE TABLE IF NOT EXISTS neighbors (
			node_id VARCHAR(128) NOT NULL, 
			hash VARCHAR(64) NOT NULL, 
			ip VARCHAR(39) NOT NULL, 
			tcp_port SMALLINT unsigned NOT NULL, 
			udp_port SMALLINT unsigned NOT NULL, 
			first_received_at DECIMAL(18,6) NOT NULL, 
			last_received_at DECIMAL(18,6) NOT NULL, 
			count BIGINT unsigned DEFAULT 1, 
			PRIMARY KEY (node_id, ip, tcp_port, udp_port)
		)
	`); err != nil {
		log.Error("Failed to create neighbors table", "database", srv.MySQLName, "err", err)
		return err
	}
	log.Trace("neighbors table already exists or is newly created", "database", srv.MySQLName)

	// create node_meta_info table
	if _, err := srv.DB.Exec(`
		CREATE TABLE IF NOT EXISTS node_meta_info (
			node_id VARCHAR(128) NOT NULL, 
			hash VARCHAR(64) NOT NULL, 
			dial_count BIGINT unsigned DEFAULT 0, 
			accept_count BIGINT unsigned DEFAULT 0, 
			p2p_disc4_count bigint unsigned default 0, 
			eth_disc4_count bigint unsigned default 0, 
			PRIMARY KEY (node_id)
		)
	`); err != nil {
		log.Error("Failed to create node_meta_info table", "database", srv.MySQLName, "err", err)
		return err
	}
	log.Trace("node_meta_info table already exists or is newly created", "database", srv.MySQLName)

	// create node_info table
	if _, err := srv.DB.Exec(`
		CREATE TABLE IF NOT EXISTS node_info (
			id BIGINT unsigned NOT NULL AUTO_INCREMENT, 
			node_id VARCHAR(128) NOT NULL, 
			ip VARCHAR(39) NOT NULL, 
			tcp_port SMALLINT unsigned NOT NULL, 
			remote_port SMALLINT unsigned NOT NULL, 
			p2p_version BIGINT unsigned NULL, 
			client_id VARCHAR(255) NULL, 
			caps VARCHAR(255) NULL, 
			listen_port SMALLINT unsigned NULL, 
			hello_count BIGINT unsigned DEFAULT 0, 
			first_hello_at DECIMAL(18,6) NULL, 
			last_hello_at DECIMAL(18,6) NULL, 
			protocol_version BIGINT unsigned NULL, 
			network_id BIGINT unsigned NULL, 
			first_received_td DECIMAL(65) NULL, 
			last_received_td DECIMAL(65) NULL, 
			best_hash VARCHAR(64) NULL, 
			genesis_hash VARCHAR(64) NULL, 
			dao_fork TINYINT NULL, 
			status_count BIGINT unsigned DEFAULT 0, 
			first_status_at DECIMAL(18,6) NULL, 
			last_status_at DECIMAL(18,6) NULL, 
			PRIMARY KEY (id), 
			KEY (node_id)
		)
	`); err != nil {
		log.Error("Failed to create node_info table", "database", srv.MySQLName, "err", err)
		return err
	}
	log.Trace("node_info table already exists or is newly created", "database", srv.MySQLName)

	return nil
}

func (srv *Server) CloseSql() {
	if srv.DB != nil {
		// close prepared sql statements
		srv.closeSqlStmts()

		// close db handle
		srv.closeDB(srv.DB)
	}
}

func (srv *Server) closeDB(db *sql.DB) {
	if err := db.Close(); err != nil {
		log.Error("Failed to close sql db handle", "database", srv.MySQLName, "err", err)
	}
	log.Trace("Closed sql db handle", "database", srv.MySQLName)
}

func (srv *Server) closeSqlStmts() {
	if srv.addNodeInfoStmt != nil {
		if err := srv.addNodeInfoStmt.Close(); err != nil {
			log.Error("Failed to close AddNodeInfo sql statement", "err", err)
		} else {
			log.Trace("Closed AddNodeInfo sql statement")
		}
	}
	if srv.updateNodeInfoStmt != nil {
		if err := srv.updateNodeInfoStmt.Close(); err != nil {
			log.Error("Failed to close UpdateNodeInfo sql statement", "err", err)
		} else {
			log.Trace("Closed UpdateNodeInfo sql statement")
		}
	}
	if srv.addNodeMetaInfoStmt != nil {
		if err := srv.addNodeMetaInfoStmt.Close(); err != nil {
			log.Error("Failed to close AddNodeMetaInfo sql statement", "err", err)
		} else {
			log.Trace("Closed AddNodeMetaInfo sql statement")
		}
	}
	if srv.GetRowIDStmt != nil {
		if err := srv.GetRowIDStmt.Close(); err != nil {
			log.Error("Failed to close GetRowID sql statement", "err", err)
		} else {
			log.Trace("Closed GetRowID sql statement")
		}
	}
}

func (srv *Server) loadKnownNodeInfos() {
	rows, _ := srv.DB.Query(`
		SELECT ni.id, ni.node_id, nmi.hash, ip, tcp_port, remote_port, 
			p2p_version, client_id, caps, listen_port, first_hello_at, last_hello_at, 
			protocol_version, network_id, first_received_td, last_received_td, best_hash, genesis_hash, 
			first_status_at, last_status_at, dao_fork 
		FROM (SELECT * 
			  FROM node_info x INNER JOIN (SELECT node_id as nid, MAX(id) as max_id 
										   FROM node_info 
										   GROUP BY node_id
										   ) max_ids ON x.id = max_ids.max_id
			  ) ni INNER JOIN node_meta_info nmi ON ni.node_id=nmi.node_id
	`)
	defer rows.Close()

	type sqlObjects struct {
		p2pVersion      sql.NullInt64
		clientId        sql.NullString
		caps            sql.NullString
		listenPort      sql.NullInt64
		firstHelloAt    sql.NullFloat64
		lastHelloAt     sql.NullFloat64
		protocolVersion sql.NullInt64
		networkId       sql.NullInt64
		firstReceivedTd sql.NullString
		lastReceivedTd  sql.NullString
		bestHash        sql.NullString
		genesisHash     sql.NullString
		daoForkSupport  sql.NullInt64
		firstStatusAt   sql.NullFloat64
		lastStatusAt    sql.NullFloat64
	}

	srv.KnownNodeInfos.Lock()
	defer srv.KnownNodeInfos.Unlock()

	for rows.Next() {
		var (
			rowId      uint64
			nodeid     string
			hash       string
			ip         string
			tcpPort    uint16
			remotePort uint16
			sqlObj     sqlObjects
		)
		err := rows.Scan(&rowId, &nodeid, &hash, &ip, &tcpPort, &remotePort,
			&sqlObj.p2pVersion, &sqlObj.clientId, &sqlObj.caps, &sqlObj.listenPort,
			&sqlObj.firstHelloAt, &sqlObj.lastHelloAt, &sqlObj.protocolVersion, &sqlObj.networkId,
			&sqlObj.firstReceivedTd, &sqlObj.lastReceivedTd, &sqlObj.bestHash, &sqlObj.genesisHash,
			&sqlObj.firstStatusAt, &sqlObj.lastStatusAt, &sqlObj.daoForkSupport)
		if err != nil {
			log.Error("Failed to copy values from sql query result", "err", err)
			continue
		}
		// convert hex to NodeID
		id, err := discover.HexID(nodeid)
		if err != nil {
			log.Error("Failed to parse node_id value from db", "rowId", rowId, "id", nodeid, "err", err)
			continue
		}
		nodeInfo := &Info{
			RowId:         rowId,
			Keccak256Hash: hash,
			IP:            ip,
			TCPPort:       tcpPort,
			RemotePort:    remotePort,
		}
		if sqlObj.p2pVersion.Valid {
			nodeInfo.P2PVersion = uint64(sqlObj.p2pVersion.Int64)
		}
		if sqlObj.clientId.Valid {
			nodeInfo.ClientId = sqlObj.clientId.String
		}
		if sqlObj.caps.Valid {
			nodeInfo.Caps = sqlObj.caps.String
		}
		if sqlObj.listenPort.Valid {
			nodeInfo.ListenPort = uint16(sqlObj.listenPort.Int64)
		}
		if sqlObj.firstHelloAt.Valid {
			i, f := math.Modf(sqlObj.firstHelloAt.Float64)
			t := time.Unix(int64(i), int64(f*1000000000))
			nodeInfo.FirstHelloAt = &UnixTime{Time: &t}
		}
		if sqlObj.lastHelloAt.Valid {
			i, f := math.Modf(sqlObj.lastHelloAt.Float64)
			t := time.Unix(int64(i), int64(f*1000000000))
			nodeInfo.LastHelloAt = &UnixTime{Time: &t}
		}
		if sqlObj.protocolVersion.Valid {
			nodeInfo.ProtocolVersion = uint64(sqlObj.protocolVersion.Int64)
		}
		if sqlObj.networkId.Valid {
			nodeInfo.NetworkId = uint64(sqlObj.networkId.Int64)
		}
		if sqlObj.firstReceivedTd.Valid {
			firstReceivedTd := &big.Int{}
			s := sqlObj.firstReceivedTd.String
			_, ok := firstReceivedTd.SetString(s, 10)
			if !ok {
				log.Error("Failed to parse first_received_td value from db", "rowId", rowId, "value", s)
			} else {
				nodeInfo.FirstReceivedTd = NewTd(firstReceivedTd)
			}
		}
		if sqlObj.lastReceivedTd.Valid {
			lastReceivedTd := &big.Int{}
			s := sqlObj.lastReceivedTd.String
			_, ok := lastReceivedTd.SetString(s, 10)
			if !ok {
				log.Error("Failed to parse last_received_td value from db", "rowId", rowId, "value", s)
			} else {
				nodeInfo.LastReceivedTd = NewTd(lastReceivedTd)
			}
		}
		if sqlObj.bestHash.Valid {
			nodeInfo.BestHash = sqlObj.bestHash.String
		}
		if sqlObj.genesisHash.Valid {
			nodeInfo.GenesisHash = sqlObj.genesisHash.String
		}
		if sqlObj.firstStatusAt.Valid {
			i, f := math.Modf(sqlObj.firstStatusAt.Float64)
			t := time.Unix(int64(i), int64(f*1000000000))
			nodeInfo.FirstStatusAt = &UnixTime{Time: &t}
		}
		if sqlObj.lastStatusAt.Valid {
			i, f := math.Modf(sqlObj.lastStatusAt.Float64)
			t := time.Unix(int64(i), int64(f*1000000000))
			nodeInfo.LastStatusAt = &UnixTime{Time: &t}
		}
		if sqlObj.daoForkSupport.Valid {
			nodeInfo.DAOForkSupport = int8(sqlObj.daoForkSupport.Int64)
		}
		srv.KnownNodeInfos.Infos()[id] = nodeInfo

		// add the node to the initial static node list
		srv.addInitialStatic(id, nodeInfo)
	}
}

func (srv *Server) prepareAddNodeInfoStmt() error {
	pStmt, err := srv.DB.Prepare(`
		INSERT INTO node_info 
			(node_id, ip, tcp_port, remote_port, 
			 p2p_version, client_id, caps, listen_port, first_hello_at, last_hello_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Error("Failed to prepare AddNodeInfo sql statement", "err", err)
		return err
	} else {
		log.Trace("Prepared AddNodeInfo sql statement")
		srv.addNodeInfoStmt = pStmt
	}
	return nil
}

func (srv *Server) addNodeInfo(newInfoWrapper *KnownNodeInfosWrapper) {
	// exit if no prepared statement
	if srv.addNodeInfoStmt == nil {
		log.Crit("No prepared statement for AddNodeInfo")
		return
	}

	nodeid := newInfoWrapper.NodeId
	newInfo := newInfoWrapper.Info
	lastHelloAt := newInfo.LastHelloAt.Float64()
	_, err := srv.addNodeInfoStmt.Exec(nodeid, newInfo.IP, newInfo.TCPPort, newInfo.RemotePort,
		newInfo.P2PVersion, newInfo.ClientId, newInfo.Caps, newInfo.ListenPort, lastHelloAt, lastHelloAt)
	if err != nil {
		log.Error("Failed to execute AddNodeInfo sql statement", "id", nodeid, "newInfo", newInfo, "err", err)
	} else {
		log.Debug("Executed AddNodeInfo sql statement", "id", nodeid, "newInfo", newInfo)
	}
}

func (srv *Server) prepareUpdateNodeInfoStmt() error {
	pStmt, err := srv.DB.Prepare(`
		UPDATE node_info 
		SET remote_port=?, last_hello_at=? 
		WHERE id=?
	`)

	if err != nil {
		log.Error("Failed to prepare UpdateNodeInfo sql statement", "err", err)
		return err
	} else {
		log.Trace("Prepared UpdateNodeInfo sql statement")
		srv.updateNodeInfoStmt = pStmt
	}
	return nil
}

func (srv *Server) updateNodeInfo(newInfoWrapper *KnownNodeInfosWrapper) {
	// exit if no prepared statement
	if srv.updateNodeInfoStmt == nil {
		log.Crit("No prepared statement for UpdateNodeInfo")
		return
	}

	nodeid := newInfoWrapper.NodeId
	newInfo := newInfoWrapper.Info
	lastHelloAt := newInfo.LastHelloAt.Float64()
	_, err := srv.updateNodeInfoStmt.Exec(newInfo.RemotePort, lastHelloAt, newInfo.RowId)
	if err != nil {
		log.Error("Failed to execute UpdateNodeInfo sql statement", "id", nodeid, "newInfo", newInfo, "err", err)
	} else {
		log.Debug("Executed UpdateNodeInfo sql statement", "id", nodeid, "newInfo", newInfo)
	}
}

func (srv *Server) prepareAddNodeMetaInfoStmt() error {
	pStmt, err := srv.DB.Prepare(`
		INSERT INTO node_meta_info (node_id, hash, dial_count, accept_count, p2p_disc4_count, eth_disc4_count) 
		VALUES (?, ?, ?, ?, ?, ?) 
		ON DUPLICATE KEY UPDATE 
		dial_count=dial_count+VALUES(dial_count), 
		accept_count=accept_count+VALUES(accept_count), 
		p2p_disc4_count=p2p_disc4_count+values(p2p_disc4_count), 
		eth_disc4_count=eth_disc4_count+values(eth_disc4_count)
	`)
	if err != nil {
		log.Error("Failed to prepare AddNodeMetaInfo sql statement", "err", err)
		return err
	} else {
		log.Trace("Prepared AddNodeMetaInfo sql statement")
		srv.addNodeMetaInfoStmt = pStmt
	}
	return nil
}

func (srv *Server) addNodeMetaInfo(nodeid string, hash string, dial bool, accept bool, tooManyPeers bool) {
	// exit if no prepared statement
	if srv.addNodeMetaInfoStmt == nil {
		log.Crit("No prepared statement for AddNodeMetaInfo")
		return
	}
	p2pDisc4, ethDisc4 := false, false
	if tooManyPeers {
		if dial || accept {
			p2pDisc4 = true
		} else {
			ethDisc4 = true
		}
	}
	_, err := srv.addNodeMetaInfoStmt.Exec(nodeid, hash, boolToInt(dial), boolToInt(accept),
		boolToInt(p2pDisc4), boolToInt(ethDisc4))
	if err != nil {
		log.Error("Failed to execute AddNodeMetaNodeInfo sql statement", "id", nodeid,
			"dial", dial, "accept", accept, "p2pDisc4", p2pDisc4, "ethDisc4", ethDisc4, "err", err)
	} else {
		log.Debug("Executed AddNodeMetaNodeInfo sql statement", "id", nodeid,
			"dial", dial, "accept", accept, "p2pDisc4", p2pDisc4, "ethDisc4", ethDisc4)
	}
}

func (srv *Server) prepareGetRowID() error {
	pStmt, err := srv.DB.Prepare(`
		SELECT MAX(id) 
		FROM node_info 
		WHERE node_id=?
	`)
	if err != nil {
		log.Error("Failed to prepare GetRowID sql statement", "err", err)
		return err
	} else {
		log.Trace("Prepared GetRowID sql statement")
		srv.GetRowIDStmt = pStmt
	}
	return nil
}

func (srv *Server) getRowID(nodeid string) uint64 {
	// exit if no prepared statement
	if srv.GetRowIDStmt == nil {
		log.Crit("No prepared statement for GetRowID")
		return 0
	}

	var rowId uint64
	err := srv.GetRowIDStmt.QueryRow(nodeid).Scan(&rowId)
	if err != nil {
		log.Error("Failed to execute GetRowID sql statement", "id", nodeid, "err", err)
		return 0
	} else {
		log.Debug("Executed GetRowID sql statement", "id", nodeid, "rowId", rowId)
		return rowId
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
