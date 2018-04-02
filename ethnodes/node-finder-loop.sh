#!/bin/bash
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: sudo ./node-finder-loop.sh instance-number"
  exit 1
fi

i=$1
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd ${WORKING_DIR}
source .env
URL="research-scan.sprai.org"
NODEFINDER_PORT=30310
PORT=$(( ${NODEFINDER_PORT}+${i} ))
MYSQL_PORT=3300
DATADIR="${ROOT_DIR}-${i}/ethereum"
mkdir -p ${DATADIR}
LOGFILE="${DATADIR}/node-finder.log"
IDENTITY="uiuc-node-finder-${i} (${URL})"
MYSQL_URL="${MYSQL_USERNAME}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:$(( ${MYSQL_PORT}+${i} )))/${MYSQL_DB}"
SLEEP=3
echo "starting node-finder-${i}..."
while true
do
  geth \
    --identity "${IDENTITY}" \
    --datadir "${DATADIR}" \
    --port ${PORT} \
    --mysql "${MYSQL_URL}" \
    --nomaxpeers \
    --verbosity 5 \
    --dialfreq 1800 \
    --maxnumfile 20480 \
    --backupsql >>${LOGFILE} 2>&1
  echo "node-finder-${i} stopped. restarting in ${SLEEP} seconds..."
  sleep ${SLEEP}
  echo "restarting node-finder-${i}..."
done
