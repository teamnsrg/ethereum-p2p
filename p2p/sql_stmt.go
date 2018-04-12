package p2p

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/teamnsrg/go-ethereum/log"
)

const (
	tableName = "node_eth_info"
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
		if result, err := srv.checkIfTableExists(dbName, tableName); err != nil {
			log.Sql(fmt.Sprintf("Failed to check if %s table exists", tableName), "database", srv.MySQLName, "err", err)
			return err
		} else if result == 0 {
			log.Sql(tableName+" table not found", "database", srv.MySQLName)
			return fmt.Errorf(tableName+" table not found at %v", srv.MySQLName)
		}
		log.Sql(tableName+" table exists", "database", srv.MySQLName)
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
