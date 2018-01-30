#!/usr/bin/python
import mysqldb

def main():
    # connect to db
    conn = mysqldb.connect()
    if not conn:
        exit(1)
    mysqldb.clear_all(conn)
    mysqldb.create_all(conn)
    conn.close()

if __name__ == "__main__":
    main()