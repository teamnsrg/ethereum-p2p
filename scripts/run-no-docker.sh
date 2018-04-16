#!/bin/bash
# This script assumes required docker images are already built.
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: sudo ./run-no-docker.sh num-instance"
  exit 1
fi

n=$(( $1 - 1 ))
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ETHMONITOR_NAME="geth-eth-monitor"

# get env variables
source .env

MYSQL_PORT=3306
cd ${WORKING_DIR}/..
make geth
cp ${WORKING_DIR}/../build/bin/geth /usr/bin/geth

cd ${WORKING_DIR}
DATADIR="${ROOT_DIR}/${ETHMONITOR_NAME}"
mkdir -p -m 755 ${DATADIR}

# run eth-monitors
for i in `seq 0 ${n}`;
do
  eth-monitor-loop.sh ${i} >>${DATADIR}/${ETHMONITOR_NAME}-${i}-loop.log 2>&1 &
  echo "${ETHMONITOR_NAME}-${i} loop started"
done
