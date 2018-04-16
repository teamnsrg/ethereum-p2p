package p2p

import (
	"database/sql"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/teamnsrg/go-ethereum/common/mticker"
	"github.com/teamnsrg/go-ethereum/log"
	"github.com/teamnsrg/go-ethereum/p2p/discover"
	"syscall"
)

type SqlStrings struct {
	Prefix    string
	Suffix    string
	Values    string
	NumValues int
}

const (
	defaultMaxSqlChunkSize = 50
	defaultMaxSqlQueueSize = 1e6
	maxNumRetry            = 10
)

var (
	neighborInfoSqlString = SqlStrings{
		Prefix: "INSERT INTO neighbors " +
			"(node_id, hash, ip, tcp_port, udp_port, first_received_at, last_received_at) VALUES ",
		Suffix:    " ON DUPLICATE KEY UPDATE last_received_at=VALUES(last_received_at), count=count+1",
		Values:    "(?, ?, ?, ?, ?, ?, ?)",
		NumValues: 7,
	}

	metaInfoSqlString = SqlStrings{
		Prefix: "INSERT INTO node_meta_info " +
			"(node_id, hash, dial_count, accept_count, p2p_disc4_count, eth_disc4_count) VALUES ",
		Suffix: " ON DUPLICATE KEY UPDATE " +
			"dial_count=dial_count+VALUES(dial_count), " +
			"accept_count=accept_count+VALUES(accept_count), " +
			"p2p_disc4_count=p2p_disc4_count+values(p2p_disc4_count), " +
			"eth_disc4_count=eth_disc4_count+values(eth_disc4_count)",
		Values:    "(?, ?, ?, ?, ?, ?)",
		NumValues: 6,
	}

	p2pInfoSqlString = SqlStrings{
		Prefix: "INSERT INTO node_p2p_info " +
			"(node_id, ip, tcp_port, remote_port, " +
			"p2p_version, client_id, caps, listen_port, first_hello_at, last_hello_at, hello_count) VALUES ",
		Suffix: " ON DUPLICATE KEY UPDATE " +
			"remote_port=IF(last_hello_at <= VALUES(last_hello_at), VALUES(remote_port), remote_port), " +
			"last_hello_at=IF(last_hello_at <= VALUES(last_hello_at), VALUES(last_hello_at), last_hello_at), " +
			"hello_count=hello_count+1",
		Values:    "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)",
		NumValues: 10,
	}

	ethInfoSqlString = SqlStrings{
		Prefix: "INSERT INTO node_eth_info (node_id, ip, tcp_port, remote_port, " +
			"p2p_version, client_id, caps, listen_port, " +
			"protocol_version, network_id, first_received_td, last_received_td, " +
			"best_hash, genesis_hash, dao_fork, first_status_at, last_status_at, status_count) VALUES ",
		Suffix: " ON DUPLICATE KEY UPDATE " +
			"remote_port=IF(last_status_at <= VALUES(last_status_at), VALUES(remote_port), remote_port), " +
			"last_received_td=IF(last_status_at <= VALUES(last_status_at), VALUES(last_received_td), last_received_td), " +
			"best_hash=IF(last_status_at <= VALUES(last_status_at), VALUES(best_hash), best_hash), " +
			"dao_fork=IF(last_status_at <= VALUES(last_status_at), VALUES(dao_fork), dao_fork), " +
			"last_status_at=IF(last_status_at <= VALUES(last_status_at), VALUES(last_status_at), last_status_at), " +
			"status_count=status_count+VALUES(status_count)",
		Values:    "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		NumValues: 18,
	}
)

type infoQueue struct {
	*sync.Mutex
	infos []interface{}
}

func (q *infoQueue) add(info interface{}) {
	q.infos = append(q.infos, info)
}

func (q *infoQueue) len() int {
	return len(q.infos)
}

func (q *infoQueue) chunks(chunkSize int) []interface{} {
	total := len(q.infos)
	if total <= chunkSize {
		return []interface{}{q.infos}
	}
	chunks := []interface{}{}
	quot := total / chunkSize
	for i := 0; i < quot; i++ {
		start := i * chunkSize
		chunks = append(chunks, q.infos[start:start+chunkSize])
	}
	return append(chunks, q.infos[quot*chunkSize:])
}

func (q *infoQueue) trunc(index int) {
	q.infos = q.infos[index:]
}

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
		srv.db = db

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

		// initialize channels and queues
		srv.NeighborChan = make(chan []interface{})
		srv.metaInfoChan = make(chan []interface{})
		srv.p2pInfoChan = make(chan []interface{})
		srv.EthInfoChan = make(chan []interface{})
		srv.neighborInfoQueue = &infoQueue{&sync.Mutex{}, make([]interface{}, 0)}
		srv.metaInfoQueue = &infoQueue{&sync.Mutex{}, make([]interface{}, 0)}
		srv.p2pInfoQueue = &infoQueue{&sync.Mutex{}, make([]interface{}, 0)}
		srv.ethInfoQueue = &infoQueue{&sync.Mutex{}, make([]interface{}, 0)}

		if srv.MaxSqlChunk <= 0 {
			srv.MaxSqlChunk = defaultMaxSqlChunkSize
		}
		if srv.MaxSqlQueue <= 0 {
			srv.MaxSqlQueue = defaultMaxSqlQueueSize
		}
		srv.loopWG.Add(1)
		go srv.dbPushLoop()
	}
	return nil
}

func (srv *Server) checkIfTableExists(dbName string, tableName string) (int, error) {
	var result int
	err := srv.db.QueryRow(`
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
		log.Sql(tableName+" table not found. Skipping backup", "database", srv.MySQLName)
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
	if _, err := srv.db.Exec(fmt.Sprintf(`
		SELECT * INTO OUTFILE '/backup/%s' FROM %s
	`, fileName, tableName)); err != nil {
		log.Sql(fmt.Sprintf("Failed to create %s", fileName), "database", srv.MySQLName, "err", err)
		return err
	}
	log.Sql(fileName+" created", "database", srv.MySQLName)
	return nil
}

func (srv *Server) dropTables() error {
	if _, err := srv.db.Exec(`
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
	if _, err := srv.db.Exec(`
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
	if _, err := srv.db.Exec(`
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
	if _, err := srv.db.Exec(`
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
	if _, err := srv.db.Exec(`
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

func (srv *Server) closeSql() {
	// make sure all sql related channels are closed
	// so that the dbPushLoop properly quits
	if srv.NeighborChan != nil {
		close(srv.NeighborChan)
	}
	if srv.metaInfoChan != nil {
		close(srv.metaInfoChan)
	}
	if srv.p2pInfoChan != nil {
		close(srv.p2pInfoChan)
	}
	if srv.EthInfoChan != nil {
		close(srv.EthInfoChan)
	}

	if srv.db == nil {
		return
	}
	if err := srv.db.Close(); err != nil {
		log.Sql("Failed to close sql db handle", "database", srv.MySQLName, "err", err)
	}
	log.Sql("Closed sql db handle", "database", srv.MySQLName)
}

func (srv *Server) loadKnownNodeInfos() error {
	rows, err := srv.db.Query(`
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
		p2pVersion      sql.NullString
		clientId        sql.NullString
		caps            sql.NullString
		listenPort      sql.NullInt64
		firstHelloAt    sql.NullFloat64
		lastHelloAt     sql.NullFloat64
		protocolVersion sql.NullString
		networkId       sql.NullString
		firstReceivedTd sql.NullString
		lastReceivedTd  sql.NullString
		bestHash        sql.NullString
		genesisHash     sql.NullString
		daoForkSupport  sql.NullInt64
		firstStatusAt   sql.NullFloat64
		lastStatusAt    sql.NullFloat64
	}

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
			s := sqlObj.p2pVersion.String
			nodeInfo.P2PVersion, err = strconv.ParseUint(s, 10, 64)
			if err != nil {
				log.Sql("Failed to parse p2p_version from db", "value", s, "err", err)
				continue
			}
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
			t := time.Unix(int64(i), int64(f*1e9))
			nodeInfo.FirstHelloAt = &UnixTime{Time: &t}
		}
		if sqlObj.lastHelloAt.Valid {
			i, f := math.Modf(sqlObj.lastHelloAt.Float64)
			t := time.Unix(int64(i), int64(f*1e9))
			nodeInfo.LastHelloAt = &UnixTime{Time: &t}
		}
		if sqlObj.protocolVersion.Valid {
			s := sqlObj.protocolVersion.String
			nodeInfo.ProtocolVersion, err = strconv.ParseUint(s, 10, 64)
			if err != nil {
				log.Sql("Failed to parse protocol_version from db", "value", s, "err", err)
				continue
			}
		}
		if sqlObj.networkId.Valid {
			s := sqlObj.networkId.String
			nodeInfo.NetworkId, err = strconv.ParseUint(s, 10, 64)
			if err != nil {
				log.Sql("Failed to parse network_id from db", "value", s, "err", err)
				continue
			}
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
			t := time.Unix(int64(i), int64(f*1e9))
			nodeInfo.FirstStatusAt = &UnixTime{Time: &t}
		}
		if sqlObj.lastStatusAt.Valid {
			i, f := math.Modf(sqlObj.lastStatusAt.Float64)
			t := time.Unix(int64(i), int64(f*1e9))
			nodeInfo.LastStatusAt = &UnixTime{Time: &t}
		}
		if sqlObj.daoForkSupport.Valid {
			nodeInfo.DAOForkSupport = int8(sqlObj.daoForkSupport.Int64)
		}
		srv.KnownNodeInfos.AddInfo(id, nodeInfo)

		// add the node to the initial static node list
		srv.addInitialStatic(id, nodeInfo)
	}
	return nil
}

func (srv *Server) dbPushLoop() {
	defer srv.loopWG.Done()
	log.Sql("Starting database update push loop")
	srv.pushTicker = mticker.NewMutableTicker(time.Duration(srv.PushFreq * float64(time.Second)))
	defer srv.pushTicker.Stop()
	stopping := false
	for {
		select {
		case <-srv.pushTicker.C:
			// insert all pending info
			if !stopping {
				log.Sql("Pushing all pending updates to database")
				srv.loopWG.Add(4)
				go srv.addNeighbors(true)
				go srv.addNodeMetaInfos(true)
				go srv.addNodeP2PInfos(true)
				go srv.addNodeEthInfos(true)
			}
		case info, ok := <-srv.NeighborChan:
			if !ok {
				log.Sql("UDP listener stopped. Pushing all pending Neighbors updates")
				srv.loopWG.Add(1)
				go srv.addNeighbors(false)
				srv.NeighborChan = nil
			} else if len(info) != neighborInfoSqlString.NumValues {
				log.Sql("Not enough values for AddNeighbors")
			} else {
				srv.neighborInfoQueue.Lock()
				srv.neighborInfoQueue.add(info)
				srv.neighborInfoQueue.Unlock()
			}
		case info, ok := <-srv.metaInfoChan:
			if !ok {
				log.Sql("P2P networking stopped. Pushing all pending NodeMetaInfo updates")
				srv.loopWG.Add(1)
				go srv.addNodeMetaInfos(false)
				srv.metaInfoChan = nil
			} else if len(info) != metaInfoSqlString.NumValues {
				log.Sql("Not enough values for AddNodeMetaInfos")
			} else {
				srv.metaInfoQueue.Lock()
				srv.metaInfoQueue.add(info)
				srv.metaInfoQueue.Unlock()
			}
		case info, ok := <-srv.p2pInfoChan:
			if !ok {
				log.Sql("P2P networking stopped. Pushing all pending NodeP2PInfo updates")
				srv.loopWG.Add(1)
				go srv.addNodeP2PInfos(false)
				srv.p2pInfoChan = nil
			} else if len(info) != p2pInfoSqlString.NumValues {
				log.Sql("Not enough values for AddNodeP2PInfos")
			} else {
				srv.p2pInfoQueue.Lock()
				srv.p2pInfoQueue.add(info)
				srv.p2pInfoQueue.Unlock()
			}
		case info, ok := <-srv.EthInfoChan:
			if !ok {
				log.Sql("Ethereum protocol stopped. Pushing all pending NodeEthInfo updates")
				srv.loopWG.Add(1)
				go srv.addNodeEthInfos(false)
				srv.EthInfoChan = nil
			} else if len(info) != ethInfoSqlString.NumValues {
				log.Sql("Not enough values for AddNodeEthInfos")
			} else {
				srv.ethInfoQueue.Lock()
				srv.ethInfoQueue.add(info)
				srv.ethInfoQueue.Unlock()
			}
		}
		if srv.NeighborChan == nil && srv.metaInfoChan == nil && srv.p2pInfoChan == nil && srv.EthInfoChan == nil {
			log.Sql("Stopping database update push loop")
			return
		}
		if !stopping {
			srv.neighborInfoQueue.Lock()
			srv.metaInfoQueue.Lock()
			srv.p2pInfoQueue.Lock()
			srv.ethInfoQueue.Lock()
			total := srv.neighborInfoQueue.len() + srv.metaInfoQueue.len() + srv.p2pInfoQueue.len() + srv.ethInfoQueue.len()
			srv.neighborInfoQueue.Unlock()
			srv.metaInfoQueue.Unlock()
			srv.p2pInfoQueue.Unlock()
			srv.ethInfoQueue.Unlock()
			if total >= srv.MaxSqlQueue {
				log.Sql("Queue size grew too big. Sending interrupt signal", "size", total)
				syscall.Kill(syscall.Getpid(), syscall.SIGINT)
				stopping = true
			}
		}
	}
}

func (srv *Server) addNeighbors(chunk bool) {
	defer srv.loopWG.Done()
	srv.neighborInfoQueue.Lock()
	defer srv.neighborInfoQueue.Unlock()
	total := srv.neighborInfoQueue.len()
	if total == 0 {
		log.Sql("No Neighbor update to push")
		return
	}
	log.Sql(fmt.Sprintf("%d Neighbor updates to push", total))
	if chunk {
		valueStrings := make([]string, 0, srv.MaxSqlChunk)
		valueArgs := make([]interface{}, 0, srv.MaxSqlChunk*neighborInfoSqlString.NumValues)
		chunks := srv.neighborInfoQueue.chunks(srv.MaxSqlChunk)
		for _, c := range chunks {
			infos := c.([]interface{})
			n := len(infos)
			if n <= 0 {
				continue
			}
			for _, infoInterface := range infos {
				info, ok := infoInterface.([]interface{})
				if !ok {
					continue
				}
				valueStrings = append(valueStrings, neighborInfoSqlString.Values)
				valueArgs = append(valueArgs, info...)
			}
			stmt := neighborInfoSqlString.Prefix + strings.Join(valueStrings, ",") + neighborInfoSqlString.Suffix
			_, err := srv.db.Exec(stmt, valueArgs...)
			if err != nil {
				log.Sql("Failed to execute AddNeighbors sql statement", "numUpdates", n, "err", err)
				return
			}
			log.Sql("Executed AddNeighbors sql statement", "numUpdates", n)
			srv.neighborInfoQueue.trunc(n)
			valueStrings = valueStrings[:0]
			valueArgs = valueArgs[:0]
		}
	} else {
		for _, infoInterface := range srv.neighborInfoQueue.infos {
			info, ok := infoInterface.([]interface{})
			if !ok {
				continue
			}
			stmt := neighborInfoSqlString.Prefix + neighborInfoSqlString.Values + neighborInfoSqlString.Suffix
			for i := 1; i <= maxNumRetry; i++ {
				_, err := srv.db.Exec(stmt, info...)
				if err == nil {
					log.Sql("Executed AddNeighbors sql statement for a single record")
					break
				} else {
					log.Sql("Failed to execute AddNeighbors sql statement for a single record", "try", i, "err", err)
					if i == maxNumRetry {
						log.Sql(fmt.Sprintf("Failed to execute AddNeighbors sql statement for a single record after %d tries. Skipping", maxNumRetry))
						break
					}
				}
			}
		}
	}
}

func (srv *Server) addNodeMetaInfos(chunk bool) {
	defer srv.loopWG.Done()
	srv.metaInfoQueue.Lock()
	defer srv.metaInfoQueue.Unlock()
	total := srv.metaInfoQueue.len()
	if total == 0 {
		log.Sql("No NodeMetaInfo update to push")
		return
	}
	log.Sql(fmt.Sprintf("%d NodeMetaInfo updates to push", total))
	if chunk {
		valueStrings := make([]string, 0, srv.MaxSqlChunk)
		valueArgs := make([]interface{}, 0, srv.MaxSqlChunk*metaInfoSqlString.NumValues)
		chunks := srv.metaInfoQueue.chunks(srv.MaxSqlChunk)
		for _, c := range chunks {
			infos := c.([]interface{})
			n := len(infos)
			if n <= 0 {
				continue
			}
			for _, infoInterface := range srv.metaInfoQueue.infos {
				info, ok := infoInterface.([]interface{})
				if !ok {
					continue
				}
				valueStrings = append(valueStrings, metaInfoSqlString.Values)
				valueArgs = append(valueArgs, info...)
			}
			stmt := metaInfoSqlString.Prefix + strings.Join(valueStrings, ",") + metaInfoSqlString.Suffix
			_, err := srv.db.Exec(stmt, valueArgs...)
			if err != nil {
				log.Sql("Failed to execute AddNodeMetaInfos sql statement", "numUpdates", n, "err", err)
				return
			}
			log.Sql("Executed AddNodeMetaInfos sql statement", "numUpdates", n)
			srv.metaInfoQueue.trunc(n)
			valueStrings = valueStrings[:0]
			valueArgs = valueArgs[:0]
		}
	} else {
		for _, infoInterface := range srv.metaInfoQueue.infos {
			info, ok := infoInterface.([]interface{})
			if !ok {
				continue
			}
			stmt := metaInfoSqlString.Prefix + metaInfoSqlString.Values + metaInfoSqlString.Suffix
			for i := 1; i <= maxNumRetry; i++ {
				_, err := srv.db.Exec(stmt, info...)
				if err == nil {
					log.Sql("Executed AddNodeMetaInfos sql statement for a single record")
					break
				} else {
					log.Sql("Failed to execute AddNodeMetaInfos sql statement for a single record", "try", i, "err", err)
					if i == maxNumRetry {
						log.Sql(fmt.Sprintf("Failed to execute AddNodeMetaInfos sql statement for a single record after %d tries. Skipping", maxNumRetry))
						break
					}
				}
			}
		}
	}
}

func (srv *Server) addNodeP2PInfos(chunk bool) {
	defer srv.loopWG.Done()
	srv.p2pInfoQueue.Lock()
	defer srv.p2pInfoQueue.Unlock()
	total := srv.p2pInfoQueue.len()
	if total == 0 {
		log.Sql("No NodeP2PInfo update to push")
		return
	}
	log.Sql(fmt.Sprintf("%d NodeP2PInfo updates to push", total))
	if chunk {
		valueStrings := make([]string, 0, srv.MaxSqlChunk)
		valueArgs := make([]interface{}, 0, srv.MaxSqlChunk*p2pInfoSqlString.NumValues)
		chunks := srv.p2pInfoQueue.chunks(srv.MaxSqlChunk)
		for _, c := range chunks {
			infos := c.([]interface{})
			n := len(infos)
			if n <= 0 {
				continue
			}
			for _, infoInterface := range srv.p2pInfoQueue.infos {
				info, ok := infoInterface.([]interface{})
				if !ok {
					continue
				}
				valueStrings = append(valueStrings, p2pInfoSqlString.Values)
				valueArgs = append(valueArgs, info...)
			}
			stmt := p2pInfoSqlString.Prefix + strings.Join(valueStrings, ",") + p2pInfoSqlString.Suffix
			_, err := srv.db.Exec(stmt, valueArgs...)
			if err != nil {
				log.Sql("Failed to execute AddNodeP2PInfos sql statement", "numUpdates", n, "err", err)
				return
			}
			log.Sql("Executed AddNodeP2PInfos sql statement", "numUpdates", n)
			srv.p2pInfoQueue.trunc(n)
			valueStrings = valueStrings[:0]
			valueArgs = valueArgs[:0]
		}
	} else {
		for _, infoInterface := range srv.p2pInfoQueue.infos {
			info, ok := infoInterface.([]interface{})
			if !ok {
				continue
			}
			stmt := p2pInfoSqlString.Prefix + p2pInfoSqlString.Values + p2pInfoSqlString.Suffix
			for i := 1; i <= maxNumRetry; i++ {
				_, err := srv.db.Exec(stmt, info...)
				if err == nil {
					log.Sql("Executed AddNodeP2PInfos sql statement for a single record")
					break
				} else {
					log.Sql("Failed to execute AddNodeP2PInfos sql statement for a single record", "try", i, "err", err)
					if i == maxNumRetry {
						log.Sql(fmt.Sprintf("Failed to execute AddNodeP2PInfos sql statement for a single record after %d tries. Skipping", maxNumRetry))
						break
					}
				}
			}
		}
	}
}

func (srv *Server) addNodeEthInfos(chunk bool) {
	defer srv.loopWG.Done()
	srv.ethInfoQueue.Lock()
	defer srv.ethInfoQueue.Unlock()
	total := srv.ethInfoQueue.len()
	if total == 0 {
		log.Sql("No NodeEthInfo update to push")
		return
	}
	log.Sql(fmt.Sprintf("%d NodeEthInfo updates to push", total))
	if chunk {
		valueStrings := make([]string, 0, srv.MaxSqlChunk)
		valueArgs := make([]interface{}, 0, srv.MaxSqlChunk*ethInfoSqlString.NumValues)
		chunks := srv.ethInfoQueue.chunks(srv.MaxSqlChunk)
		for _, c := range chunks {
			infos := c.([]interface{})
			n := len(infos)
			if n <= 0 {
				continue
			}
			for _, infoInterface := range srv.ethInfoQueue.infos {
				info, ok := infoInterface.([]interface{})
				if !ok {
					continue
				}
				valueStrings = append(valueStrings, ethInfoSqlString.Values)
				valueArgs = append(valueArgs, info...)
			}
			stmt := ethInfoSqlString.Prefix + strings.Join(valueStrings, ",") + ethInfoSqlString.Suffix
			_, err := srv.db.Exec(stmt, valueArgs...)
			if err != nil {
				log.Sql("Failed to execute AddNodeEthInfos sql statement", "numUpdates", n, "err", err)
				return
			}
			log.Sql("Executed AddNodeEthInfos sql statement", "numUpdates", n)
			srv.ethInfoQueue.trunc(n)
			valueStrings = valueStrings[:0]
			valueArgs = valueArgs[:0]
		}
	} else {
		for _, infoInterface := range srv.ethInfoQueue.infos {
			info, ok := infoInterface.([]interface{})
			if !ok {
				continue
			}
			stmt := ethInfoSqlString.Prefix + ethInfoSqlString.Values + ethInfoSqlString.Suffix
			for i := 1; i <= maxNumRetry; i++ {
				_, err := srv.db.Exec(stmt, info...)
				if err == nil {
					log.Sql("Executed AddNodeEthInfos sql statement for a single record")
					break
				} else {
					log.Sql("Failed to execute AddNodeEthInfos sql statement for a single record", "try", i, "err", err)
					if i == maxNumRetry {
						log.Sql(fmt.Sprintf("Failed to execute AddNodeEthInfos sql statement for a single record after %d tries. Skipping", maxNumRetry))
						break
					}
				}
			}
		}
	}
}

func (srv *Server) queueNodeMetaInfo(id discover.NodeID, hash string, dial bool, accept bool, tooManyPeers bool) {
	p2pDisc4, ethDisc4 := false, false
	if tooManyPeers {
		if dial || accept {
			p2pDisc4 = true
		} else {
			ethDisc4 = true
		}
	}
	srv.metaInfoChan <- []interface{}{
		id.String(), hash,
		boolToInt(dial), boolToInt(accept),
		boolToInt(p2pDisc4), boolToInt(ethDisc4),
	}
}

func (srv *Server) queueNodeP2PInfo(id discover.NodeID, newInfo *Info) error {
	if newInfo.FirstHelloAt == nil {
		return fmt.Errorf("FirstHelloAt == nil")
	}
	if newInfo.LastHelloAt == nil {
		return fmt.Errorf("LastHelloAt == nil")
	}
	srv.p2pInfoChan <- []interface{}{
		id.String(), newInfo.IP, newInfo.TCPPort, newInfo.RemotePort,
		newInfo.P2PVersion, newInfo.ClientId, newInfo.Caps, newInfo.ListenPort,
		newInfo.FirstHelloAt.Float64(), newInfo.LastHelloAt.Float64(),
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
