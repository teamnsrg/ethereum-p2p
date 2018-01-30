#!/usr/bin/python
import pymysql
import time
import config

HOST = config.mysql['host']
PORT = config.mysql['port']
USER = config.mysql['user']
PASSWORD = config.mysql['password']
DB = config.mysql['db']

def clear_all(conn):
    with conn.cursor() as cursor:
        sql = "DROP TABLE IF EXISTS node_info, node_meta_info, neighbors"
        cursor.execute(sql)
        conn.commit()
    print("tables dropped!")

def create_all(conn):
    create_neighbors(conn)
    create_node_meta_info(conn)
    create_node_info(conn)

def connect(host=HOST, port=PORT, user=USER, passwd=PASSWORD, db=DB):
    if "CHANGEME" in host:
        print("please change CHANGEME in config.py")
        return None
    try:
        connection = pymysql.connect(host=host,port=port,user=user, passwd=passwd, db=db, charset="utf8mb4")
        try:
            with connection.cursor() as cursor:
                cursor.execute("SELECT VERSION()")
                result = cursor.fetchone()
            # Check if anything at all is returned
            if not result:
                print("connected to database, but something is wrong. host={}, port={}, db={}, user={}, version='unknown'".format(host, port, db, user))
            else:
                print("connected to database. host={}, port={}, db={}, user={}, version={}".format(host, port, db, user, result[0]))
        finally:
            return connection
    except pymysql.Error as error:
        print("could not connect to database. host={}, port={}, db={}, user={}".format(host, port, db, user))
    return None

def create_neighbors(conn):
    with conn.cursor() as cursor:
        sql = "CREATE TABLE IF NOT EXISTS neighbors (" \
              "node_id VARCHAR(128) NOT NULL, " \
              "ip VARCHAR(39) NOT NULL, " \
              "tcp_port SMALLINT unsigned NOT NULL, " \
              "udp_port SMALLINT unsigned NOT NULL, " \
              "first_received_at DECIMAL(18,6) NULL, " \
              "last_received_at DECIMAL(18,6) NULL, " \
              "count BIGINT unsigned DEFAULT 1, " \
              "PRIMARY KEY (node_id, ip, tcp_port, udp_port)" \
              ")"
        cursor.execute(sql)
        conn.commit()
    print("neighbors table created!")

def create_node_meta_info(conn):
    with conn.cursor() as cursor:
        sql = "CREATE TABLE IF NOT EXISTS node_meta_info (" \
              "node_id VARCHAR(128) NOT NULL, " \
              "hash VARCHAR(64) NOT NULL, " \
              "dial_count BIGINT unsigned DEFAULT 0, " \
              "accept_count BIGINT unsigned DEFAULT 0, " \
              "too_many_peers_count BIGINT unsigned DEFAULT 0, " \
              "PRIMARY KEY (node_id)" \
              ")"
        cursor.execute(sql)
        conn.commit()
    print("node_meta_info table created!")

def create_node_info(conn):
    with conn.cursor() as cursor:
        sql = "CREATE TABLE IF NOT EXISTS node_info (" \
              "id BIGINT unsigned NOT NULL AUTO_INCREMENT, " \
              "node_id VARCHAR(128) NOT NULL, " \
              "ip VARCHAR(39) NOT NULL, " \
              "tcp_port SMALLINT unsigned NOT NULL, " \
              "remote_port SMALLINT unsigned NOT NULL, " \
              "p2p_version TINYINT unsigned NOT NULL, " \
              "client_id VARCHAR(255) NOT NULL, " \
              "caps VARCHAR(255) NOT NULL, " \
              "listen_port SMALLINT unsigned NOT NULL, " \
              "protocol_version BIGINT unsigned NOT NULL, " \
              "network_id BIGINT unsigned NOT NULL, " \
              "first_received_td DECIMAL(65) NOT NULL, " \
              "last_received_td DECIMAL(65) NOT NULL, " \
              "best_hash VARCHAR(64) NOT NULL, " \
              "genesis_hash VARCHAR(64) NOT NULL, " \
              "dao_fork TINYINT unsigned NULL, " \
              "first_received_at DECIMAL(18,6) NULL, " \
              "last_received_at DECIMAL(18,6) NULL, " \
              "PRIMARY KEY (id), " \
              "KEY (node_id)" \
              ")"
        cursor.execute(sql)
        conn.commit()
    print("node_info table created!")

def main():
    conn = connect(HOST, PORT, USER, PASSWORD, DB)
    while conn is None:
        time.sleep(3)
        conn = connect(HOST, PORT, USER, PASSWORD, DB)
    create_all(conn)
    conn.close()

if __name__ == "__main__":
    main()
