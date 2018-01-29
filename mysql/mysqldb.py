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
        sql = "DROP TABLE IF EXISTS eth_status, devp2p_hello, node_id_hash, neighbors"
        cursor.execute(sql)
        conn.commit()
    print("tables dropped!")

def create_all(conn):
    create_neighbors(conn)
    create_node_id_hash(conn)
    create_devp2p_hello(conn)
    create_eth_status(conn)

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
              "first_ts DECIMAL(12) NULL, " \
              "last_ts DECIMAL(12) NULL, " \
              "count BIGINT unsigned DEFAULT 1, " \
              "PRIMARY KEY (node_id, ip, tcp_port, udp_port)" \
              ")"
        cursor.execute(sql)
        conn.commit()
    print("neighbors table created!")

def create_node_id_hash(conn):
    with conn.cursor() as cursor:
        sql = "CREATE TABLE IF NOT EXISTS node_id_hash (" \
              "node_id VARCHAR(128) NOT NULL, " \
              "hash VARCHAR(64) NOT NULL, " \
              "PRIMARY KEY (node_id)" \
              ")"
        cursor.execute(sql)
        conn.commit()
    print("node_id_hash table created!")

def create_devp2p_hello(conn):
    with conn.cursor() as cursor:
        sql = "CREATE TABLE IF NOT EXISTS devp2p_hello (" \
              "node_id VARCHAR(128) NOT NULL, " \
              "ip VARCHAR(39) NOT NULL, " \
              "tcp_port SMALLINT unsigned NOT NULL, " \
              "remote_port SMALLINT unsigned NOT NULL, " \
              "p2p_version TINYINT unsigned NOT NULL, " \
              "client_id VARCHAR(255) NOT NULL, " \
              "caps VARCHAR(255) NOT NULL, " \
              "listen_port SMALLINT unsigned NOT NULL, " \
              "first_ts DECIMAL(12) NULL, " \
              "last_ts DECIMAL(12) NULL, " \
              "dial_count BIGINT unsigned DEFAULT 1, " \
              "accept_count BIGINT unsigned DEFAULT 1, " \
              "PRIMARY KEY (node_id, ip, tcp_port, p2p_version, client_id, caps, listen_port)" \
              ")"
        cursor.execute(sql)
        conn.commit()
    print("devp2p_hello table created!")

def create_eth_status(conn):
    with conn.cursor() as cursor:
        sql = "CREATE TABLE IF NOT EXISTS eth_status (" \
              "node_id VARCHAR(128) NOT NULL, " \
              "ip VARCHAR(39) NOT NULL, " \
              "tcp_port SMALLINT unsigned NOT NULL, " \
              "protocol_version BIGINT unsigned NOT NULL, " \
              "network_id BIGINT unsigned NOT NULL, " \
              "td DECIMAL(65) NOT NULL, " \
              "best_hash VARCHAR(64) NOT NULL, " \
              "block_number DECIMAL(24) NOT NULL, " \
              "genesis_hash VARCHAR(64) NOT NULL, " \
              "dao_fork TINYINT unsigned NULL, " \
              "first_ts DECIMAL(12) NULL, " \
              "last_ts DECIMAL(12) NULL, " \
              "count BIGINT unsigned DEFAULT 1, " \
              "PRIMARY KEY (node_id, ip, tcp_port, protocol_version, network_id, td, best_hash, block_number, genesis_hash)" \
              ")"
        cursor.execute(sql)
        conn.commit()
    print("eth_status table created!")

def main():
    conn = connect(HOST, PORT, USER, PASSWORD, DB)
    while conn is None:
        time.sleep(3)
        conn = connect(HOST, PORT, USER, PASSWORD, DB)
    create_all(conn)
    conn.close()

if __name__ == "__main__":
    main()
