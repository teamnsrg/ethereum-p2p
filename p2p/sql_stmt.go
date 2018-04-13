package p2p

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/teamnsrg/go-ethereum/common/mticker"
	"github.com/teamnsrg/go-ethereum/log"
	"github.com/teamnsrg/go-ethereum/p2p/discover"
	"net"
	"time"
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
		srv.db = db

		// check if table exists
		sqlNameParsed := strings.Split(srv.MySQLName, "/")
		dbName := sqlNameParsed[len(sqlNameParsed)-1]
		for _, tableName := range []string{"node_meta_info", "node_eth_info"} {
			if result, err := srv.checkIfTableExists(dbName, tableName); err != nil {
				log.Sql(fmt.Sprintf("Failed to check if %s table exists", tableName), "database", srv.MySQLName, "err", err)
				return err
			} else if result == 0 {
				log.Sql(tableName+" table not found", "database", srv.MySQLName)
				return fmt.Errorf(tableName+" table not found at %v", srv.MySQLName)
			}
			log.Sql(tableName+" table exists", "database", srv.MySQLName)
		}
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

func (srv *Server) closeSql() {
	if srv.db == nil {
		return
	}
	if err := srv.db.Close(); err != nil {
		log.Sql("Failed to close sql db handle", "database", srv.MySQLName, "err", err)
	}
	log.Sql("Closed sql db handle", "database", srv.MySQLName)
}

func (srv *Server) queryNodeAddress(interval time.Duration) error {
	rows, err := srv.db.Query(`
		SELECT new_addrs.node_id, ip, tcp_port 
		FROM node_meta_info 
			INNER JOIN 
				(SELECT all_addrs.node_id, ip, tcp_port 
				 FROM (SELECT node_id, MAX(last_status_at) AS last_status_at 
				 	   FROM node_eth_info 
				 	   WHERE last_status_at >= ? 
				 	   GROUP BY node_id
				 	   ) AS new_ids 
				 	INNER JOIN 
				 		(SELECT node_id, ip, tcp_port, last_status_at 
				 		 FROM node_eth_info
				 		 ) AS all_addrs 
				 		ON new_ids.last_status_at = all_addrs.last_status_at
				 ) AS new_addrs 
				ON new_addrs.node_id = node_meta_info.node_id 
		WHERE dial_count > 0
	`, float64(time.Now().Add(-interval).UnixNano())/1e9)
	defer rows.Close()
	if err != nil {
		log.Sql("Failed to execute node address query", "database", srv.MySQLName, "err", err)
		return err
	}
	log.Sql("Executed node address query", "database", srv.MySQLName)

	var nodes []*discover.Node
	for rows.Next() {
		var (
			nodeid  string
			ip      net.IP
			host    string
			tcpPort uint16
		)
		err := rows.Scan(&nodeid, &host, &tcpPort)
		if err != nil {
			log.Sql("Failed to copy values from sql query result", "err", err)
			continue
		}
		if tcpPort == 0 {
			log.Sql("Node is not added to the static node list because its TCP port is 0", "id", nodeid, "ip", host)
			continue
		}
		// convert hex to NodeID
		id, err := discover.HexID(nodeid)
		if err != nil {
			log.Sql("Failed to parse node_id value from db", "id", nodeid, "err", err)
			continue
		}
		if ip = net.ParseIP(host); ip == nil {
			log.Sql("Failed to parse ip from db", "id", nodeid, "ip", host, "err", err)
			continue
		}

		nodes = append(nodes, discover.NewNode(id, ip, tcpPort, tcpPort))
	}
	// add nodes to static list at once to avoid flooding the srv.addstatic channel
	srv.AddPeers(nodes)
	return nil
}

func (srv *Server) dbNodeQueryLoop() {
	defer srv.loopWG.Done()
	log.Sql("Starting node address query loop")
	// initial query
	srv.queryNodeAddress(time.Duration(srv.RedialExp * float64(time.Hour)))

	var interval time.Duration
	if srv.QueryFreq > 0.0 {
		interval = time.Duration(srv.QueryFreq * float64(time.Second))
	} else {
		// if queryTicker is 0, then set the ticker to the default value just to keep the ticker running
		interval = defaultQueryFreq
	}
	srv.queryTicker = mticker.NewMutableTicker(interval)
	defer srv.queryTicker.Stop()
	for {
		select {
		case <-srv.queryTicker.C:
			srv.queryNodeAddress(interval)
		case _, ok := <-srv.quit:
			if !ok {
				log.Sql("P2P networking stopped. Stopping node address query loop")
				return
			}
		}
	}
}
