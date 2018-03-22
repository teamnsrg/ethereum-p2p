package p2p

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/teamnsrg/go-ethereum/log"
)

const (
	tableName = "node_info"
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

		// check if table exists
		sqlNameParsed := strings.Split(srv.MySQLName, "/")
		dbName := sqlNameParsed[len(sqlNameParsed)-1]
		if result, err := srv.checkIfTableExists(dbName, tableName); err != nil {
			log.Error(fmt.Sprintf("Failed to check if %s table exists", tableName), "database", srv.MySQLName, "err", err)
			return err
		} else if result == 0 {
			return fmt.Errorf(tableName+" table not found at %v", srv.MySQLName)
		}
		log.Trace(tableName+" table exists", "database", srv.MySQLName)
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

func (srv *Server) CloseSql() {
	if srv.DB != nil {
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
