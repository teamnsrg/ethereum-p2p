#!/bin/bash
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: sudo ./eth-monitor-loop.sh instance-number"
  exit 1
fi

i=$1
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd ${WORKING_DIR}
source .env
MYSQL_PORT=3306
ETHMONITOR_NAME="geth-eth-monitor"
URL="research-scan.sprai.org"
ETHMONITOR_PORT=30303
PORT=$(( ${ETHMONITOR_PORT}+${i} ))
DATADIR="${ROOT_DIR}/${ETHMONITOR_NAME}/${i}"
mkdir -p -m 755 ${DATADIR}/${ETHMONITOR_NAME}
ERRFILE="${DATADIR}/${ETHMONITOR_NAME}-error.log"
IDENTITY="uiuc-${i}(${URL})"
MYSQL_URL="${MYSQL_USERNAME}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DB}"
SLEEP=3
echo "starting ${ETHMONITOR_NAME}-${i}..."
while true
do
  geth \
    --identity "${IDENTITY}" \
    --datadir "${DATADIR}" \
    --port ${PORT} \
    --verbosity 5 \
    --mysql "${MYSQL_URL}" \
    --logtofile \
    --queryfreq 180 \
    --redialfreq 60 \
    --redialcheckfreq 3 \
    --redialexp 24 \
    --maxnumfile 1048576 \
    --maxredial 100 >>${ERRFILE} 2>&1
  echo "${ETHMONITOR_NAME}-${i} stopped. restarting in ${SLEEP} seconds..."
  sleep ${SLEEP}
  echo "restarting ${ETHMONITOR_NAME}-${i}..."
done
