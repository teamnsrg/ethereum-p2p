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
		log.Sql("No sql db connection info provided")
	} else {
		db, err := sql.Open("mysql", srv.MySQLName)
		if err != nil {
			log.Sql("Failed to open sql db handle", "database", srv.MySQLName, "err", err)
			return err
		}
		log.Sql("Opened sql db handle", "database", srv.MySQLName)
		if err := db.Ping(); err != nil {
			log.Sql("Sql db connection failed ping test", "database", srv.MySQLName, "err", err)
			return err
		}
		log.Sql("Sql db connection passed ping test", "database", srv.MySQLName)
		srv.DB = db

		// backup tables
		// If ResetSQL is true, BackupSQL should be true as well
		if srv.BackupSQL {
			sqlNameParsed := strings.Split(srv.MySQLName, "/")
			dbName := sqlNameParsed[len(sqlNameParsed)-1]
			for _, tableName := range []string{"neighbors", "node_meta_info", "node_p2p_info", "node_eth_info"} {
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
		if err := srv.loadKnownNodeInfos(); err != nil {
			return err
		}

		// prepare sql statements
		if err := srv.prepareAddNodeP2PInfoStmt(); err != nil {
			return err
		}
		if err := srv.prepareAddNodeMetaInfoStmt(); err != nil {
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
		log.Sql(fmt.Sprintf("Failed to check if %s table exists", tableName), "database", srv.MySQLName, "err", err)
		return err
	} else if result == 0 {
		log.Debug(tableName+" table not found. Skipping backup", "database", srv.MySQLName)
		return nil
	}
	log.Sql(tableName+" table exists", "database", srv.MySQLName)

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
		log.Sql(fmt.Sprintf("Failed to create %s", fileName), "database", srv.MySQLName, "err", err)
		return err
	}
	log.Sql(fileName+" created", "database", srv.MySQLName)
	return nil
}

func (srv *Server) dropTables() error {
	if _, err := srv.DB.Exec(`
		DROP TABLE IF EXISTS node_eth_info, node_p2p_info, node_meta_info, neighbors
	`); err != nil {
		log.Sql("Failed to drop sql tables", "database", srv.MySQLName, "err", err)
		return err
	}
	log.Sql("Existing sql tables dropped", "database", srv.MySQLName)
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
		log.Sql("Failed to create neighbors table", "database", srv.MySQLName, "err", err)
		return err
	}
	log.Sql("neighbors table already exists or is newly created", "database", srv.MySQLName)

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
		log.Sql("Failed to create node_meta_info table", "database", srv.MySQLName, "err", err)
		return err
	}
	log.Sql("node_meta_info table already exists or is newly created", "database", srv.MySQLName)

	// create node_p2p_info table
	if _, err := srv.DB.Exec(`
		CREATE TABLE IF NOT EXISTS node_p2p_info (
			node_id VARCHAR(128) NOT NULL, 
			ip VARCHAR(39) NOT NULL, 
			tcp_port SMALLINT unsigned NOT NULL, 
			remote_port SMALLINT unsigned NOT NULL, 
			p2p_version BIGINT unsigned NOT NULL, 
			client_id VARCHAR(255) NOT NULL, 
			caps VARCHAR(255) NOT NULL, 
			listen_port SMALLINT unsigned NOT NULL, 
			hello_count BIGINT unsigned DEFAULT 0, 
			first_hello_at DECIMAL(18,6) NOT NULL, 
			last_hello_at DECIMAL(18,6) NOT NULL, 
			PRIMARY KEY (node_id, ip, tcp_port, p2p_version, client_id, caps, listen_port),
			KEY (last_hello_at)
		)
	`); err != nil {
		log.Sql("Failed to create node_p2p_info table", "database", srv.MySQLName, "err", err)
		return err
	}
	log.Sql("node_p2p_info table already exists or is newly created", "database", srv.MySQLName)

	// create node_eth_info table
	if _, err := srv.DB.Exec(`
		CREATE TABLE IF NOT EXISTS node_eth_info (
			node_id VARCHAR(128) NOT NULL, 
			ip VARCHAR(39) NOT NULL, 
			tcp_port SMALLINT unsigned NOT NULL, 
			remote_port SMALLINT unsigned NOT NULL, 
			p2p_version BIGINT unsigned NOT NULL, 
			client_id VARCHAR(255) NOT NULL, 
			caps VARCHAR(255) NOT NULL, 
			listen_port SMALLINT unsigned NOT NULL, 
			protocol_version BIGINT unsigned NOT NULL, 
			network_id BIGINT unsigned NOT NULL, 
			first_received_td DECIMAL(65) NOT NULL, 
			last_received_td DECIMAL(65) NOT NULL, 
			best_hash VARCHAR(64) NOT NULL, 
			genesis_hash VARCHAR(64) NOT NULL, 
			dao_fork TINYINT NULL, 
			status_count BIGINT unsigned DEFAULT 0, 
			first_status_at DECIMAL(18,6) NOT NULL, 
			last_status_at DECIMAL(18,6) NOT NULL, 
			PRIMARY KEY (node_id, ip, tcp_port, p2p_version, client_id, caps, listen_port, protocol_version, network_id, genesis_hash), 
			KEY (last_status_at)
		)
	`); err != nil {
		log.Sql("Failed to create node_eth_info table", "database", srv.MySQLName, "err", err)
		return err
	}
	log.Sql("node_eth_info table already exists or is newly created", "database", srv.MySQLName)
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
		log.Sql("Failed to close sql db handle", "database", srv.MySQLName, "err", err)
	}
	log.Sql("Closed sql db handle", "database", srv.MySQLName)
}

func (srv *Server) closeSqlStmts() {
	if srv.addNodeP2PInfoStmt != nil {
		if err := srv.addNodeP2PInfoStmt.Close(); err != nil {
			log.Sql("Failed to close AddNodeInfo sql statement", "err", err)
		} else {
			log.Sql("Closed AddNodeInfo sql statement")
		}
	}
	if srv.addNodeMetaInfoStmt != nil {
		if err := srv.addNodeMetaInfoStmt.Close(); err != nil {
			log.Sql("Failed to close AddNodeMetaInfo sql statement", "err", err)
		} else {
			log.Sql("Closed AddNodeMetaInfo sql statement")
		}
	}
}

func (srv *Server) loadKnownNodeInfos() error {
	rows, err := srv.DB.Query(`
		SELECT nmi.node_id, nmi.hash, ip, tcp_port, remote_port, 
			p2p_version, client_id, caps, listen_port, first_hello_at, last_hello_at, 
			protocol_version, network_id, first_received_td, last_received_td, best_hash, genesis_hash, 
			first_status_at, last_status_at, dao_fork 
		FROM node_meta_info AS nmi 
			INNER JOIN 
				(SELECT p2p.node_id, p2p.ip, p2p.tcp_port, p2p.remote_port, p2p.p2p_version, 
						p2p.client_id, p2p.caps, p2p.listen_port, p2p.first_hello_at, p2p.last_hello_at, 
						eth.protocol_version, eth.network_id, eth.first_received_td, eth.last_received_td, 
						eth.best_hash, eth.genesis_hash, eth.first_status_at, eth.last_status_at, eth.dao_fork  
				 FROM (SELECT * 
				 	   FROM node_p2p_info AS x 
				 	   		INNER JOIN 
				 	   			(SELECT node_id AS nid, MAX(last_hello_at) AS last_ts 
				 	   			 FROM node_p2p_info 
				 	   			 GROUP BY node_id
				 	   			 ) AS last_tss 
				 	   			ON x.last_hello_at = last_tss.last_ts
				 	   ) AS p2p 
				 	LEFT JOIN 
				 	  (SELECT * 
					   FROM node_eth_info AS x 
					   		INNER JOIN 
					   			(SELECT node_id AS nid, MAX(last_status_at) AS last_ts 
							  	 FROM node_eth_info 
							  	 GROUP BY node_id
							  	 ) AS last_tss 
								ON x.last_status_at = last_tss.last_ts
					   ) AS eth 
						ON p2p.node_id = eth.node_id
				 ) AS nodes
					ON nmi.node_id = nodes.node_id
	`)
	if err != nil {
		log.Sql("Failed to execute initial query", "database", srv.MySQLName, "err", err)
		return err
	}
	log.Sql("Executed initial query", "database", srv.MySQLName)
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
			nodeid     string
			hash       string
			ip         string
			tcpPort    uint16
			remotePort uint16
			sqlObj     sqlObjects
		)
		err := rows.Scan(&nodeid, &hash, &ip, &tcpPort, &remotePort,
			&sqlObj.p2pVersion, &sqlObj.clientId, &sqlObj.caps, &sqlObj.listenPort,
			&sqlObj.firstHelloAt, &sqlObj.lastHelloAt, &sqlObj.protocolVersion, &sqlObj.networkId,
			&sqlObj.firstReceivedTd, &sqlObj.lastReceivedTd, &sqlObj.bestHash, &sqlObj.genesisHash,
			&sqlObj.firstStatusAt, &sqlObj.lastStatusAt, &sqlObj.daoForkSupport)
		if err != nil {
			log.Sql("Failed to copy values from sql query result", "err", err)
			continue
		}
		// convert hex to NodeID
		id, err := discover.HexID(nodeid)
		if err != nil {
			log.Sql("Failed to parse node_id value from db", "id", nodeid, "err", err)
			continue
		}
		nodeInfo := &Info{
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
				log.Sql("Failed to parse first_received_td value from db", "value", s)
			} else {
				nodeInfo.FirstReceivedTd = NewTd(firstReceivedTd)
			}
		}
		if sqlObj.lastReceivedTd.Valid {
			lastReceivedTd := &big.Int{}
			s := sqlObj.lastReceivedTd.String
			_, ok := lastReceivedTd.SetString(s, 10)
			if !ok {
				log.Sql("Failed to parse last_received_td value from db", "value", s)
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
	return nil
}

func (srv *Server) prepareAddNodeP2PInfoStmt() error {
	pStmt, err := srv.DB.Prepare(`
		INSERT INTO node_p2p_info 
			(node_id, ip, tcp_port, remote_port, 
			 p2p_version, client_id, caps, listen_port, first_hello_at, last_hello_at, hello_count) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1) 
		ON DUPLICATE KEY UPDATE 
			remote_port=VALUES(remote_port), 
			last_hello_at=VALUES(last_hello_at), 
			hello_count=hello_count+1
	`)
	if err != nil {
		log.Sql("Failed to prepare AddNodeP2PInfo sql statement", "err", err)
		return err
	} else {
		log.Sql("Prepared AddNodeP2PInfo sql statement")
		srv.addNodeP2PInfoStmt = pStmt
	}
	return nil
}

func (srv *Server) addNodeP2PInfo(newInfoWrapper *KnownNodeInfosWrapper) {
	// exit if no prepared statement
	if srv.addNodeP2PInfoStmt == nil {
		log.Crit("No prepared statement for AddNodeP2PInfo")
		return
	}

	nodeid := newInfoWrapper.NodeId
	newInfo := newInfoWrapper.Info
	lastHelloAt := newInfo.LastHelloAt.Float64()
	_, err := srv.addNodeP2PInfoStmt.Exec(nodeid, newInfo.IP, newInfo.TCPPort, newInfo.RemotePort,
		newInfo.P2PVersion, newInfo.ClientId, newInfo.Caps, newInfo.ListenPort, lastHelloAt, lastHelloAt)
	if err != nil {
		log.Sql("Failed to execute AddNodeP2PInfo sql statement", "id", nodeid[:16], "err", err)
	} else {
		log.Sql("Executed AddNodeP2PInfo sql statement", "id", nodeid[:16])
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
		log.Sql("Failed to prepare AddNodeMetaInfo sql statement", "err", err)
		return err
	} else {
		log.Sql("Prepared AddNodeMetaInfo sql statement")
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
		log.Sql("Failed to execute AddNodeMetaNodeInfo sql statement", "id", nodeid[:16], "dial", dial, "accept", accept, "p2pDisc4", p2pDisc4, "ethDisc4", ethDisc4, "err", err)
	} else {
		log.Sql("Executed AddNodeMetaNodeInfo sql statement", "id", nodeid[:16], "dial", dial, "accept", accept, "p2pDisc4", p2pDisc4, "ethDisc4", ethDisc4)
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
