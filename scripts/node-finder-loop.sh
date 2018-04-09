#!/bin/bash
# check if root
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
MYSQL_PORT=3306
NODEFINDER_NAME="geth-node-finder"
URL="research-scan.sprai.org"
NODEFINDER_PORT=30310
PORT=$(( ${NODEFINDER_PORT}+${i} ))
DATADIR="${ROOT_DIR}/${NODEFINDER_NAME}/${i}"
mkdir -p -m 755 ${DATADIR}/${NODEFINDER_NAME}
ERRFILE="${DATADIR}/${NODEFINDER_NAME}-error.log"
IDENTITY="uiuc-${i}(${URL})"
MYSQL_URL="${MYSQL_USERNAME}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DB}"
SLEEP=3
echo "starting ${NODEFINDER_NAME}-${i}..."
while true
do
  geth \
    --identity "${IDENTITY}" \
    --datadir "${DATADIR}" \
    --port ${PORT} \
    --verbosity 5 \
    --mysql "${MYSQL_URL}" \
    --logtofile \
    --redialfreq 1800 \
    --redialcheckfreq 5 \
    --maxnumfile 20480 \
    --maxredial 1000 \
    --pushfreq 1 >>${ERRFILE} 2>&1
  echo "${NODEFINDER_NAME}-${i} stopped. restarting in ${SLEEP} seconds..."
  sleep ${SLEEP}
  echo "restarting ${NODEFINDER_NAME}-${i}..."
done
