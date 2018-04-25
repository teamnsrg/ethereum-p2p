#!/usr/bin/python3
import os,sys
import pyasn

if len(sys.argv) < 4:
    print("not enough arguments")
    print("usage: python3 finalize-node-info.py asn-db-filename node-info-filename output-filename")
    sys.exit(1)

dbfile = sys.argv[1]
if not os.path.isfile(dbfile):
    print("asn db file does not exist")
    sys.exit(1)
ASES = pyasn.pyasn(dbfile)
infile = sys.argv[2]
if not os.path.isfile(infile):
    print("node info file does not exist")
    sys.exit(1)
outfile = sys.argv[3]

with open(outfile, "w") as out:
    inf = open(infile)
    for line in inf:
        info = line.strip().split('|')
        n = len(info)
        if n == 5:
            nodeid, ip, conn, first, last = info
        elif n == 8:
            nodeid, ip, conn1, first1, last1, conn2, first2, last2 = info
            conn = "{}-{}".format(conn1, conn2)
            first = first1 if float(first1) < float(first2) else first2
            last = last1 if float(last1) > float(last2) else last2
        else:
            continue
        asn = ASES.lookup(ip)[0]
        asn = str(asn) if asn else str(0)
        out.write("{}\n".format("|".join([nodeid, ip, asn, conn, first, last])))
    inf.close()
sys.exit(0)
