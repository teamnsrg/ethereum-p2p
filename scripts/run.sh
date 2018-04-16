#!/bin/bash
# This script assumes required docker images are already built.
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: sudo ./run.sh num-instance"
  exit 1
fi

n=$(( $1 - 1 ))
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ETHMONITOR_NAME="geth-eth-monitor"

# get env variables
source .env

# run mysql container
MYSQL_PORT=3306

ETHMONITOR_IMAGE="geth:eth-monitor"

# run eth-monitors
URL="research-scan.sprai.org"
ETHMONITOR_PORT=30303
DATADIR="/root/.ethereum"
echo "starting ${ETHMONITOR_NAME} containers..."
for i in `seq 0 ${n}`;
do
  IDENTITY="uiuc-${i}(${URL})"
  ETHMONITOR_DIR="${ROOT_DIR}/${ETHMONITOR_NAME}/${i}"
  [ -d "${ETHMONITOR_DIR}" ] || mkdir -p -m 755 ${ETHMONITOR_DIR}/${ETHMONITOR_NAME}
  MYSQL_URL="${MYSQL_USERNAME}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DB}"
  PORT=$(( ${ETHMONITOR_PORT}+${i} ))
  CMD="geth \
    --identity \"${IDENTITY}\" \
    --datadir \"${DATADIR}\" \
    --port ${PORT} \
    --verbosity 5 \
    --mysql \"${MYSQL_URL}\" \
    --logtofile \
    --queryfreq 180 \
    --redialfreq 1800 \
    --redialexp 24 \
    --maxnumfile 1048576 \
    --maxredial 1000"
  docker run -dit --restart=always -h ${ETHMONITOR_NAME}-${i} --name ${ETHMONITOR_NAME}-${i} --net host -v ${ETHMONITOR_DIR}:${DATADIR} -e CMD="${CMD}" --entrypoint '/bin/sh' ${ETHMONITOR_IMAGE} -c "${CMD}"
 echo "${ETHMONITOR_NAME}-${i} started"
done
