#!/bin/bash
# check if root
if [ "$EUID" -ne 0 ]; then
    echo "please run as root"
    exit 1
fi

pip2 install pymysql 

WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd ${WORKING_DIR}/ethnodes
docker-compose up -d

cd ${WORKING_DIR}
python2 mysqldb.py 
